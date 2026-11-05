package routes

import "github.com/gin-gonic/gin"

type WeatherHandler interface {
	GetCurrentWeather(c *gin.Context)
	GetWeatherForecast(c *gin.Context)
}

func registerWeatherRoutes(authRoutes gin.IRoutes, h WeatherHandler) {
	authRoutes.GET("/weather/current", h.GetCurrentWeather)
	authRoutes.GET("/weather/forecast", h.GetWeatherForecast)
}
