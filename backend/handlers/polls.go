package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"livepoll/middleware"
	"livepoll/models"
)

var voterIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{16,64}$`)

// Atomic: record the voter (SADD) and bump the counter only if they hadn't voted.
var voteScript = redis.NewScript(`
if redis.call('SADD', KEYS[1], ARGV[1]) == 0 then return 0 end
redis.call('HINCRBY', KEYS[2], ARGV[2], 1)
return 1
`)

func countsKey(id primitive.ObjectID) string  { return "poll:" + id.Hex() + ":counts" }
func votersKey(id primitive.ObjectID) string  { return "poll:" + id.Hex() + ":voters" }
func channelKey(id primitive.ObjectID) string { return "poll:" + id.Hex() + ":events" }

type results struct {
	Counts []int64 `json:"counts"`
	Total  int64   `json:"total"`
	Open   bool    `json:"open"`
}

type pollView struct {
	ID        string     `json:"id"`
	Question  string     `json:"question"`
	Options   []string   `json:"options"`
	Counts    []int64    `json:"counts"`
	Total     int64      `json:"total"`
	Open      bool       `json:"open"`
	Closed    bool       `json:"closed"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func isOpen(p *models.Poll) bool {
	return !p.Closed && (p.ExpiresAt == nil || time.Now().Before(*p.ExpiresAt))
}

func sum(xs []int64) (t int64) {
	for _, x := range xs {
		t += x
	}
	return
}

func toView(p *models.Poll, counts []int64) pollView {
	opts := make([]string, len(p.Options))
	for i, o := range p.Options {
		opts[i] = o.Text
	}
	return pollView{
		ID: p.ID.Hex(), Question: p.Question, Options: opts, Counts: counts,
		Total: sum(counts), Open: isOpen(p), Closed: p.Closed,
		ExpiresAt: p.ExpiresAt, CreatedAt: p.CreatedAt,
	}
}

// counts reads live counters from Redis. If the key is gone (eviction/restart)
// it rebuilds them from the durable copy in MongoDB.
func (h *Handler) counts(ctx context.Context, p *models.Poll) []int64 {
	out := make([]int64, len(p.Options))
	m, err := h.Redis.HGetAll(ctx, countsKey(p.ID)).Result()
	if err == nil && len(m) > 0 {
		for i := range out {
			if v, e := strconv.ParseInt(m[strconv.Itoa(i)], 10, 64); e == nil {
				out[i] = v
			}
		}
		return out
	}
	args := []any{}
	for i, o := range p.Options {
		out[i] = o.Votes
		if o.Votes > 0 {
			args = append(args, strconv.Itoa(i), o.Votes)
		}
	}
	if len(args) > 0 {
		h.Redis.HSet(ctx, countsKey(p.ID), args...)
	}
	return out
}

func (h *Handler) results(ctx context.Context, p *models.Poll) results {
	c := h.counts(ctx, p)
	return results{Counts: c, Total: sum(c), Open: isOpen(p)}
}

func (h *Handler) findPoll(ctx context.Context, idHex string) (*models.Poll, error) {
	id, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return nil, mongo.ErrNoDocuments
	}
	var p models.Poll
	if err := h.Polls.FindOne(ctx, bson.M{"_id": id}).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (h *Handler) loadPoll(c *gin.Context) (*models.Poll, bool) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	p, err := h.findPoll(ctx, c.Param("id"))
	if errors.Is(err, mongo.ErrNoDocuments) {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return nil, false
	}
	return p, true
}

func (h *Handler) publish(ctx context.Context, p *models.Poll) {
	payload, _ := json.Marshal(h.results(ctx, p))
	if err := h.Redis.Publish(ctx, channelKey(p.ID), payload).Err(); err != nil {
		log.Printf("publish failed: %v", err)
	}
}

// ---------- create / read / manage ----------

func (h *Handler) CreatePoll(c *gin.Context) {
	var req struct {
		Question         string   `json:"question" binding:"required"`
		Options          []string `json:"options" binding:"required"`
		ExpiresInMinutes int      `json:"expiresInMinutes" binding:"min=0,max=43200"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll data"})
		return
	}
	q := strings.TrimSpace(req.Question)
	if n := utf8.RuneCountInString(q); n < 3 || n > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question must be 3-200 characters"})
		return
	}
	if len(req.Options) < 2 || len(req.Options) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide 2 to 10 options"})
		return
	}
	seen := map[string]bool{}
	opts := make([]models.Option, 0, len(req.Options))
	for _, raw := range req.Options {
		t := strings.TrimSpace(raw)
		if n := utf8.RuneCountInString(t); n < 1 || n > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each option must be 1-100 characters"})
			return
		}
		k := strings.ToLower(t)
		if seen[k] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "options must be unique"})
			return
		}
		seen[k] = true
		opts = append(opts, models.Option{Text: t})
	}
	owner, err := primitive.ObjectIDFromHex(c.GetString(middleware.UserIDKey))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	now := time.Now().UTC()
	p := models.Poll{ID: primitive.NewObjectID(), Question: q, Options: opts, OwnerID: owner, CreatedAt: now}
	if req.ExpiresInMinutes > 0 {
		t := now.Add(time.Duration(req.ExpiresInMinutes) * time.Minute)
		p.ExpiresAt = &t
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if _, err := h.Polls.InsertOne(ctx, p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create poll"})
		return
	}
	c.JSON(http.StatusCreated, toView(&p, make([]int64, len(opts))))
}

func (h *Handler) GetPoll(c *gin.Context) {
	p, ok := h.loadPoll(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toView(p, h.counts(c.Request.Context(), p)))
}

func (h *Handler) MyPolls(c *gin.Context) {
	owner, err := primitive.ObjectIDFromHex(c.GetString(middleware.UserIDKey))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	cur, err := h.Polls.Find(ctx, bson.M{"owner_id": owner},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(100))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	var polls []models.Poll
	if err := cur.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	out := make([]pollView, 0, len(polls))
	for i := range polls {
		counts := make([]int64, len(polls[i].Options))
		for j, o := range polls[i].Options {
			counts[j] = o.Votes
		}
		out = append(out, toView(&polls[i], counts))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ownedFilter(c *gin.Context) (bson.M, bool) {
	id, err1 := primitive.ObjectIDFromHex(c.Param("id"))
	owner, err2 := primitive.ObjectIDFromHex(c.GetString(middleware.UserIDKey))
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return nil, false
	}
	return bson.M{"_id": id, "owner_id": owner}, true
}

func (h *Handler) ClosePoll(c *gin.Context) {
	f, ok := h.ownedFilter(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	res, err := h.Polls.UpdateOne(ctx, f, bson.M{"$set": bson.M{"closed": true}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if res.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return
	}
	if p, err := h.findPoll(ctx, c.Param("id")); err == nil {
		h.publish(ctx, p) // viewers see it close live
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeletePoll(c *gin.Context) {
	f, ok := h.ownedFilter(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	res, err := h.Polls.DeleteOne(ctx, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if res.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return
	}
	id := f["_id"].(primitive.ObjectID)
	h.Redis.Del(ctx, countsKey(id), votersKey(id))
	c.Status(http.StatusNoContent)
}

// ---------- voting + realtime ----------

func (h *Handler) Vote(c *gin.Context) {
	var req struct {
		OptionIndex *int   `json:"optionIndex" binding:"required"`
		VoterID     string `json:"voterId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !voterIDRe.MatchString(req.VoterID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vote"})
		return
	}
	p, ok := h.loadPoll(c)
	if !ok {
		return
	}
	idx := *req.OptionIndex
	if idx < 0 || idx >= len(p.Options) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid option"})
		return
	}
	if !isOpen(p) {
		c.JSON(http.StatusConflict, gin.H{"error": "this poll is closed"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	h.counts(ctx, p) // make sure Redis is hydrated before incrementing

	added, err := voteScript.Run(ctx, h.Redis,
		[]string{votersKey(p.ID), countsKey(p.ID)}, req.VoterID, strconv.Itoa(idx)).Int()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record vote"})
		return
	}
	if added == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "you have already voted in this poll"})
		return
	}
	// Durable copy in MongoDB.
	if _, err := h.Polls.UpdateByID(ctx, p.ID,
		bson.M{"$inc": bson.M{fmt.Sprintf("options.%d.votes", idx): 1}}); err != nil {
		log.Printf("mongo vote persist failed: %v", err)
	}
	h.publish(ctx, p)
	c.JSON(http.StatusOK, h.results(ctx, p))
}

// Stream pushes live results over Server-Sent Events, fed by Redis pub/sub.
func (h *Handler) Stream(c *gin.Context) {
	p, ok := h.loadPoll(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	sub := h.Redis.Subscribe(ctx, channelKey(p.ID))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "realtime unavailable"})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	snapshot, _ := json.Marshal(h.results(ctx, p))
	fmt.Fprintf(c.Writer, "data: %s\n\n", snapshot)
	c.Writer.Flush()

	ch := sub.Channel()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		case msg, ok := <-ch:
			if !ok {
				return false
			}
			fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
			return true
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n") // keeps proxies from closing idle streams
			return true
		}
	})
}
