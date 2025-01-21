package routes

import (
	"RoyDental/cache"
	"RoyDental/config"
	"RoyDental/controllers"
	"RoyDental/handlers"
	"RoyDental/middlewares"
	"RoyDental/repositories"
	"RoyDental/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRoutes initializes the routes and middleware for the server
func SetupRoutes(cache *cache.Cache, config *config.AppConfig, db *gorm.DB) http.Handler {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Create a Gin router
	router := gin.Default()

	// Apply Bearer token validation to all routes
	router.Use(middlewares.ValidateBearerToken(config.GetBearerToken()))

	// Create and apply CORS middleware configuration
	corsConfig := &middlewares.CorsConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "https://www.example.com", "https://example-dev.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}
	router.Use(middlewares.CorsMiddleware(corsConfig))

	// Apply rate limiter middleware
	router.Use(middlewares.NewRateLimiterMiddleware(middlewares.RateLimiterConfig{
		RequestsPerSecond: 15, // 15 requests per second
		Burst:             30, // Burst of 30
	}))

	// Apply logging middleware
	router.Use(middlewares.LoggingMiddleware())

	// Initialize repositories, services, and handlers
	emergencyContactRepo := repositories.NewEmergencyContactRepository(cache)
	billingRepo := repositories.NewBillingRepository(cache)
	examinationRepo := repositories.NewExaminationRepository(cache)
	treatmentPlanRepo := repositories.NewTreatmentPlanRepository(cache)
	appointmentRepo := repositories.NewAppointmentRepository(cache)

	patientRepo := repositories.NewPatientRepository(
		cache,
		emergencyContactRepo,
		billingRepo,
		examinationRepo,
		treatmentPlanRepo,
		appointmentRepo,
	)

	userRepo := repositories.NewUserRepository(db, cache)

	patientService := services.NewPatientService(patientRepo)
	userService := services.NewUserService(userRepo)

	patientHandler := handlers.NewPatientHandler(patientService)
	authHandler := handlers.NewAuthHandler(userService)
	doctorHandler := handlers.NewDoctorHandler(services.NewDoctorService(repositories.NewDoctorRepository(cache)))
	insuranceCompanyHandler := handlers.NewInsuranceCompanyHandler(services.NewInsuranceCompanyService(repositories.NewInsuranceCompanyRepository(cache)))
	emergencyContactHandler := handlers.NewEmergencyContactHandler(services.NewEmergencyContactService(emergencyContactRepo))
	examinationHandler := handlers.NewExaminationHandler(services.NewExaminationService(examinationRepo))
	billingHandler := handlers.NewBillingHandler(services.NewBillingService(billingRepo))
	treatmentPlanHandler := handlers.NewTreatmentPlanHandler(services.NewTreatmentPlanService(treatmentPlanRepo))
	appointmentHandler := handlers.NewAppointmentHandler(services.NewAppointmentService(appointmentRepo))

	// Register routes
	controllers.SetupPatientRoutes(
		router,
		patientHandler,
		doctorHandler,
		insuranceCompanyHandler,
		emergencyContactHandler,
		examinationHandler,
		billingHandler,
		treatmentPlanHandler,
		appointmentHandler,
	)

	authController := controllers.NewAuthController(authHandler)
	authController.RegisterRoutes(router)

	controllers.SetupRootRoute(router)

	return router
}
