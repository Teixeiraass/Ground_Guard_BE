package handler

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
)

const (
	openWeatherBase  = "https://api.openweathermap.org/data/2.5/"
	weatherCacheTTL  = 10 * time.Minute
	maxCityLength    = 100
	maxWeatherBody   = 1 << 20
	weatherCacheSize = 2000
)

type weatherEntry struct {
	status    int
	body      []byte
	expiresAt time.Time
}

// weatherProxy keeps the OpenWeather key on the server (it used to ship
// inside the app) and caches answers, so many users in the same city cost
// one upstream call every 10 minutes.
type weatherProxy struct {
	apiKey string
	client *http.Client
	mu     sync.Mutex
	cache  map[string]weatherEntry
}

func newWeatherProxy(apiKey string) *weatherProxy {
	return &weatherProxy{
		apiKey: apiKey,
		client: &http.Client{Timeout: 8 * time.Second},
		cache:  make(map[string]weatherEntry),
	}
}

func validCity(city string) bool {
	if city == "" || len(city) > maxCityLength {
		return false
	}
	for _, r := range city {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (p *weatherProxy) fetch(ctx context.Context, endpoint, city string) (int, []byte, error) {
	key := endpoint + "|" + strings.ToLower(city)
	p.mu.Lock()
	if e, ok := p.cache[key]; ok && time.Now().Before(e.expiresAt) {
		p.mu.Unlock()
		return e.status, e.body, nil
	}
	p.mu.Unlock()

	q := url.Values{}
	q.Set("q", city)
	q.Set("appid", p.apiKey)
	q.Set("units", "metric")
	q.Set("lang", "pt_br")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openWeatherBase+endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return 0, nil, err
	}
	rsp, err := p.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer rsp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(rsp.Body, maxWeatherBody))
	if err != nil {
		return 0, nil, err
	}

	if rsp.StatusCode == http.StatusOK || rsp.StatusCode == http.StatusNotFound {
		p.mu.Lock()
		if len(p.cache) >= weatherCacheSize {
			p.cache = make(map[string]weatherEntry)
		}
		p.cache[key] = weatherEntry{status: rsp.StatusCode, body: body, expiresAt: time.Now().Add(weatherCacheTTL)}
		p.mu.Unlock()
	}
	return rsp.StatusCode, body, nil
}

func (server *Server) serveWeather(ctx *gin.Context, endpoint string) {
	if server.weather.apiKey == "" {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "weather is not configured (OPENWEATHER_API_KEY)"})
		return
	}
	city := strings.TrimSpace(ctx.Query("city"))
	if !validCity(city) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "city is required"})
		return
	}

	status, body, err := server.weather.fetch(ctx, endpoint, city)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "weather provider unavailable"})
		return
	}
	switch status {
	case http.StatusOK:
		ctx.Data(http.StatusOK, "application/json; charset=utf-8", body)
	case http.StatusNotFound:
		ctx.JSON(http.StatusNotFound, gin.H{"error": "city not found"})
	default:
		// Never forward upstream errors: they can mention the API key.
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "weather provider unavailable"})
	}
}

// GetCurrentWeather godoc
// @Summary      Clima atual da cidade
// @Description  Repasse autenticado da API do OpenWeather (/weather, unidades métricas, pt_br), com cache de 10 minutos. A chave fica só no servidor.
// @Tags         weather
// @Produce      json
// @Security     BearerAuth
// @Param        city query string true "Cidade"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      502 {object} map[string]interface{}
// @Router       /weather/current [get]
func (server *Server) GetCurrentWeather(ctx *gin.Context) {
	server.serveWeather(ctx, "weather")
}

// GetWeatherForecast godoc
// @Summary      Previsão do tempo da cidade
// @Description  Repasse autenticado da API do OpenWeather (/forecast, unidades métricas, pt_br), com cache de 10 minutos.
// @Tags         weather
// @Produce      json
// @Security     BearerAuth
// @Param        city query string true "Cidade"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      502 {object} map[string]interface{}
// @Router       /weather/forecast [get]
func (server *Server) GetWeatherForecast(ctx *gin.Context) {
	server.serveWeather(ctx, "forecast")
}
