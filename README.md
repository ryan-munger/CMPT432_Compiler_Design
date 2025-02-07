# Gopiler by Ryan Munger

Dear Alan: grade the main branch -- I am using feature branches. You can use either web or CLI mode as they are functionally equivalent (only output medium differs). You will probably opt for the CLI version but I still learned A TON whilst making the web version. Go is cool!


# Setup
1. Get GoLang from your package manager or their website.
1. Ensure `go version` works.
1. Do not worry about dependencies as go run will handle everything.
1. **Gopiler has two modes of operation:**
   1. Web Mode
   2. CLI Mode

# Running Gopiler in CLI Mode
1. Note: commands to follow are to be run from the project directory.
2. **To compile and run (recommended):** `go run ./cmd/cli/main.go -v -f <filename>` 
    1. The -f arg provides the source file to compile.
    2. -v toggles verbose mode to true (to see detailed output).
    3. As always, -h or -help will provide this information.
3. To compile an executable to run yourself:
    1. You can create a bin folder. Or be messy if you want.
    2. Linux: `go build -o ./bin/gopiler ./cmd/cli/main.go`
        1. Then: `./bin/gopiler -v -f <filename>`
    3. Windows: `go build -o ./bin/compiler.exe ./cmd/cli/main.go`
        1. Then: `.\bin\gopiler.exe -v -f <filename>`

# Running Gopiler in Web Mode
1. This creates a frontend for the compiler! Using go made this VERY easy!!!
1. Note: commands to follow are to be run from the project directory.
2. Start webserver: `go run ./cmd/web/main.go`
   1. Add -e (expose) if you wish to open server to the internet (instead of localhost)
   2. -h will show information
3. A POST request will send the user's code to the backend and get back the compiler log messages!
![GUI](./Labs/images/gui.png)

# In this course I strive to:
* Gain and demonstrate an understanding of the fundamental areas of compiler
architecture: front end, intermediate representation, and the back end.
* Gain and demonstrate an understanding of context-free grammars and their use.
* Gain and demonstrate an understanding of the techniques for scanning (lexical
analysis), parsing a grammar, translation, and simple code generation.
* Embrace the opportunity to develop a complex system over the course of the
semester where I have to either live with my prior mistakes and shortcuts or go
back and fix them. (Either will teach valuable lessons.) 
* Learn that developing the software is only half the battle, debugging and testing are
critical skills for a talented professional, and skills that will be valuable. 
* Gain and demonstrate an understanding that the chasm between programs that
work once and programs that work every time is ridiculously huge.
* Enhance my continuing education skills. Capable problem solvers never stop
learning. 
