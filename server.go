package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"os"

	job "github.com/Nguyen-Hoa/job"
	worker "github.com/Nguyen-Hoa/worker"
	"github.com/gin-gonic/gin"
)

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
		if err != nil {
			c.JSON(500, err.Error())
		} else {
			c.JSON(200, stats)
		}
	})

	r.GET("/reduced-stats", func(c *gin.Context) {
		stats, err := g_worker.ReducedStats()
		if err != nil {
			c.JSON(500, err.Error())
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
		job := job.Job{}
		if err := c.BindJSON(&job); err != nil {
			log.Print(err)
			c.JSON(400, err.Error())
		}

		Id, err := g_worker.StartJob(job.Image, job.Cmd, job.Duration)
		if err != nil {
			c.JSON(500, err.Error())
		}
		c.JSON(200, Id)
	})

	r.POST("/migrate", func(c *gin.Context) {
		// stop
		// save
		// copy
		// done
	})

	r.POST("/kill", func(c *gin.Context) {
		// get container
		job := job.Job{}
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
		var containers map[string]interface{} = make(map[string]interface{})
		if len(containerStats) > 0 {
			for id, stats := range containerStats {
				if err != nil {
					log.Fatal(err)
					c.JSON(500, "Failed to read stats for container")
				} else {
					var ctrStat = make(map[string]interface{})
					json.Unmarshal(stats, &ctrStat)
					containers[id] = ctrStat
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

	r.GET("/has-power-meter", func(c *gin.Context) {
		if !g_worker.HasPowerMeter {
			c.JSON(200, "Power meter is not running")
		} else {
			c.JSON(200, "meter running")
		}
	})

	r.Run()
}
