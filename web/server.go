package web

import (
	"fmt"
	"gopiler/internal"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Structs for JSON request/response
type CompileRequest struct {
	Code string `json:"code"`
}

type CompileResponse struct {
	Output string `json:"output"`
}

func StartServer(expose bool) {
	r := gin.Default()
	r.SetTrustedProxies(nil) // Disable trusting any proxies

	// Serve HTML templates
	r.LoadHTMLGlob("web/templates/*")

	// Serve static files (JS, CSS, etc.)
	r.Static("/static", "./web/static")

	// Serve the main page
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Handle compilation requests
	r.POST("/compile", func(c *gin.Context) {
		var request struct {
			Code    string `json:"code"`
			Verbose bool   `json:"verbose"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		output := runCompiler(request.Code, request.Verbose)
		c.JSON(http.StatusOK, gin.H{"output": output})
	})

	r.GET("/getCST", func(c *gin.Context) {
		cst := internal.GetCst()
		c.String(http.StatusOK, cst)
	})

	r.GET("/getAST", func(c *gin.Context) {
		ast := internal.GetAst()
		c.String(http.StatusOK, ast)
	})

	// for symbol table display box
	r.GET("/getSymbolTables", func(c *gin.Context) {
		symbolTables := internal.GetSymbolTables()
		c.JSON(http.StatusOK, symbolTables)
	})

	// for machine code display box
	r.GET("/getMachineCode/:program", func(c *gin.Context) {
		programStr := c.Param("program")
		program, err := strconv.Atoi(programStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid program number")
			return
		}

		machineCode := internal.GetMachineCode(program, false)
		c.String(http.StatusOK, machineCode)
	})

	// for machine code display box
	r.GET("/getAssembly/:program", func(c *gin.Context) {
		programStr := c.Param("program")
		program, err := strconv.Atoi(programStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid program number")
			return
		}

		machineCode := internal.GetAssembly(program)
		c.String(http.StatusOK, machineCode)
	})

	if expose {
		log.Println("Web server exposed to internet; [host_ip]:8080")
		r.Run("0.0.0.0:8080") // exposed to internet
	} else {
		log.Println("Web server running on localhost; 0.0.0.0:8080")
		r.Run(":8080") // localhost
	}
}

func runCompiler(code string, verbose bool) string {
	internal.ResetAll()
	var output string

	internal.SetVerbose(verbose)

	internal.Info(fmt.Sprintf("Starting compilation with verbose mode: %t", internal.Verbose), "GOPILER", true)

	if len(code) == 0 {
		internal.Warn("No code provided. No compilation will be executed.", "GOPILER")
	} else {
		internal.Lex(code)
	}

	internal.Info("All compilations complete.", "GOPILER", true)
	output = internal.GetLogOutput() // retrieve logs

	return output
}
