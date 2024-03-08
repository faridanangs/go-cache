package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

func main() {
	app := fiber.New()
	api := NewApi()

	app.Get("/api", api.Handler)

	app.Listen(":8000")
}

// func (a *Api) Handler(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("in handler")
// 	q := r.URL.Query().Get("q")
// 	data, isCache, err := a.getData(q, r.Context())
// 	if err != nil {
// 		fmt.Printf("error getting data from handler %v: %v", q, err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
// 	resp := APIResponse{
// 		Cache: isCache,
// 		Data:  data,
// 	}
// 	w.WriteHeader(http.StatusOK)
// 	w.Header().Add("Content-Type", "application/json")
// 	err = json.NewEncoder(w).Encode(&resp)
// 	if err != nil {
// 		fmt.Printf("error encoding response: %v", err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
// }

func (a *Api) Handler(c *fiber.Ctx) error {
	q := c.Query("q")
	data, isCache, err := a.getData(q, context.Background())
	if err != nil {
		fmt.Printf("error getting data from handler %v: %v", q, err)
		return c.JSON(fiber.Map{
			"err": err,
		})
	}
	resp := APIResponse{
		Cache: isCache,
		Data:  data,
	}
	return c.Status(200).JSON(resp)
}

func (a *Api) getData(q string, ctx context.Context) ([]NominatinResponse, bool, error) {
	// is query cached?
	value, err := a.cache.Get(ctx, q).Result()
	if err == redis.Nil {
		escaped0 := url.PathEscape(q)

		var addr string
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr = fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json", escaped0)
		}()
		wg.Wait()

		resp, err := http.Get(addr)
		if err != nil {
			return nil, false, err
		}

		data := make([]NominatinResponse, 0)

		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, false, err
		}

		byt, err := json.Marshal(&data)
		if err != nil {
			return nil, false, err
		}

		if err := a.cache.Set(ctx, q, bytes.NewBuffer(byt).Bytes(), time.Second*10).Err(); err != nil {
			return nil, false, err
		}

		// rturn the response
		return data, false, nil
	} else if err != nil {
		fmt.Printf("error calling redis %v:", err)
		return nil, false, err
	} else {
		// build the response
		data := make([]NominatinResponse, 0)
		if err := json.Unmarshal(bytes.NewBufferString(value).Bytes(), &data); err != nil {
			return nil, false, err
		}
		// return the response
		return data, true, nil
	}
}

type Api struct {
	cache *redis.Client
}

func NewApi() *Api {
	redisAddr := fmt.Sprintf("%s:6379", os.Getenv("REDIS_URL"))

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return &Api{
		cache: rdb,
	}
}

type APIResponse struct {
	Cache bool                `json:"cache"`
	Data  []NominatinResponse `json:"data"`
}

type NominatinResponse struct {
	PlaceID     int      `json:"place_id"`
	Licence     string   `json:"licence"`
	OsmType     string   `json:"osm_type"`
	OsmID       int      `json:"osm_id"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	Class       string   `json:"class"`
	Type        string   `json:"type"`
	PlaceRank   int      `json:"place_rank"`
	Importance  float64  `json:"importance"`
	Addresstype string   `json:"addresstype"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Boundingbox []string `json:"boundingbox"`
}
