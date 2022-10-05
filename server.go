package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"os"

	worker "github.com/Nguyen-Hoa/worker"
	"github.com/gin-gonic/gin"
)

type Job struct {
	Image    string   `json:"image"`
	Cmd      []string `json:"cmd"`
	Duration int      `json:"duration"`
}

func main() {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		log.Fatal("Failed to parse configuration file.")
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var config worker.WorkerConfig
	json.Unmarshal([]byte(byteValue), &config)
	if config.RPCServer {
		worker := worker.RPCServerWorker{}
		worker.Init(config)
		worker.Available = true
		rpc.Register(&worker)
		rpc.HandleHTTP()
		if err := http.ListenAndServe(worker.RPCPort, nil); err != nil {
			log.Print(err)
		}
	} else {
		worker := worker.ServerWorker{}
		worker.Init(config)
		worker.Available = true
		runHTTPServer(worker)
	}
}

func runHTTPServer(g_worker worker.ServerWorker) {
	r := gin.Default()
	r.GET("/stats", func(c *gin.Context) {
		stats, err := g_worker.Stats()
		// body, err := json.Marshal(stats)
		if err != nil {
			c.JSON(500, "Failed to retrieve stats")
		} else {
			c.JSON(200, stats)
		}
	})

	r.POST("/meter-start", func(c *gin.Context) {
		if err := g_worker.StartMeter(); err != nil {
			c.JSON(500, gin.H{
				"message": err,
			})
			c.Abort()
		} else {
			c.JSON(200, gin.H{
				"message": "Power meter started",
			})
		}
	})

	r.POST("/meter-stop", func(c *gin.Context) {
		if err := g_worker.StopMeter(); err != nil {
			c.JSON(500, gin.H{
				"message": err,
			})
			c.Abort()
		}
		c.JSON(200, gin.H{
			"message": "Power meter stopped",
		})
	})

	r.POST("/execute", func(c *gin.Context) {
		job := Job{}
		if err := c.BindJSON(&job); err != nil {
			c.JSON(400, "Failed to parse container")
		}

		if err := g_worker.StartJob(job.Image, job.Cmd, job.Duration); err != nil {
			panic(err)
		}
		c.JSON(200, "Job started")
	})

	r.POST("/migrate", func(c *gin.Context) {
		// stop
		// save
		// copy
		// done
	})

	r.POST("/kill", func(c *gin.Context) {
		// get container
		job := Job{}
		if err := c.BindJSON(&job); err != nil {
			c.JSON(400, "Failed to parse container")
		}

		if err := g_worker.StopJob(job.Image); err != nil {
			log.Print(err)
			c.JSON(400, "Failed to stop job")
		} else {
			c.JSON(200, "Job stopped")
		}
	})

	r.GET("/running_jobs", func(c *gin.Context) {
		containers, err := g_worker.GetRunningJobs()
		if err != nil {
			log.Println(err)
			c.JSON(500, "Failed to get running jobs.")
		}
		c.JSON(200, containers)
	})

	r.GET("/running_jobs_stats", func(c *gin.Context) {
		containerStats, err := g_worker.GetRunningJobsStats()
		if err != nil {
			log.Println(err)
			c.JSON(500, "Failed to get running jobs stats")
		}
		var containers map[string]string = make(map[string]string)
		if len(containerStats) > 0 {
			for id, stats := range containerStats {
				defer stats.Body.Close()
				bytes, err := io.ReadAll(stats.Body)
				if err != nil {
					log.Fatal(err)
					c.JSON(500, "Failed to read stats for container")
				} else {
					containers[id] = string(bytes)
				}
			}
		}
		c.JSON(200, containers)
	})

	r.GET("/available", func(c *gin.Context) {
		if !g_worker.IsAvailable() {
			c.JSON(401, "Worker is not unavailable")
		} else {
			c.JSON(200, "Worker is ready to send/receive jobs.")
		}
	})

	r.Run()
}
