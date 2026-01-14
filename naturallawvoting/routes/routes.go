package routes

import (
	"voting-api/database"
	"voting-api/handlers"
	"voting-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(db *database.DB) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	ballotHandler := handlers.NewBallotHandler(db)
	voteHandler := handlers.NewVoteHandler(db)
	profileHandler := handlers.NewProfileHandler(db)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Public routes (no authentication required)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Public ballot routes (read-only)
		public := api.Group("/public")
		{
			public.GET("/ballots", ballotHandler.GetAllBallots)
			public.GET("/ballots/:id", ballotHandler.GetBallot)
			public.GET("/ballots/:id/results", voteHandler.GetBallotResults)
		}

		// Protected routes (authentication required)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// User profile
			protected.GET("/profile", authHandler.GetProfile)

			// User's ballots
			protected.GET("/my-ballots", ballotHandler.GetUserBallots)

			// Ballot management
			protected.POST("/ballots", ballotHandler.CreateBallot)

			// Voting
			protected.POST("/ballots/:ballot_id/vote", voteHandler.Vote)
			protected.GET("/ballots/:ballot_id/my-vote", voteHandler.GetUserVote)

			// Profile information routes
			// User Profile
			protected.GET("/profile/info", profileHandler.GetUserProfile)
			protected.POST("/profile/info", profileHandler.CreateUserProfile)
			protected.PUT("/profile/info", profileHandler.UpdateUserProfile)
			protected.DELETE("/profile/info", profileHandler.DeleteUserProfile)

			// User Address
			protected.GET("/profile/address", profileHandler.GetUserAddress)
			protected.POST("/profile/address", profileHandler.CreateUserAddress)
			protected.PUT("/profile/address", profileHandler.UpdateUserAddress)
			protected.DELETE("/profile/address", profileHandler.DeleteUserAddress)

			// User Political Affiliation
			protected.GET("/profile/political", profileHandler.GetUserPoliticalAffiliation)
			protected.POST("/profile/political", profileHandler.CreateUserPoliticalAffiliation)
			protected.PUT("/profile/political", profileHandler.UpdateUserPoliticalAffiliation)
			protected.DELETE("/profile/political", profileHandler.DeleteUserPoliticalAffiliation)

			// User Religious Affiliation
			protected.GET("/profile/religious", profileHandler.GetUserReligiousAffiliation)
			protected.POST("/profile/religious", profileHandler.CreateUserReligiousAffiliation)
			protected.PUT("/profile/religious", profileHandler.UpdateUserReligiousAffiliation)
			protected.DELETE("/profile/religious", profileHandler.DeleteUserReligiousAffiliation)

			// User Race/Ethnicity
			protected.GET("/profile/race-ethnicity", profileHandler.GetUserRaceEthnicity)
			protected.POST("/profile/race-ethnicity", profileHandler.CreateUserRaceEthnicity)
			protected.PUT("/profile/race-ethnicity", profileHandler.UpdateUserRaceEthnicity)
			protected.DELETE("/profile/race-ethnicity", profileHandler.DeleteUserRaceEthnicity)

			// Economic Info
			protected.GET("/profile/economic", profileHandler.GetEconomicInfo)
			protected.POST("/profile/economic", profileHandler.CreateEconomicInfo)
			protected.PUT("/profile/economic", profileHandler.UpdateEconomicInfo)
			protected.DELETE("/profile/economic", profileHandler.DeleteEconomicInfo)
		}
	}

	return r
}