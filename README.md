# Gopiler by Ryan Munger

Dear Alan: grade the main branch


# Running Mr. Gopiler
1. Get GoLang from your package manager or their website.
1. Ensure `go version` works.
1. The following commands are to be run from the project directory. I will be using the standardized GoLang project structure. 
1. **To compile and run (recommended):** `go run ./cmd/compiler/ -f <filename>` 
    1. Currently, the -f arg is just how I will take in the filename to compile.
1. To compile an executable to run yourself:
    1. Linux: `go build -o ./bin/compiler ./cmd/compiler`
        1. Then: `./bin/compiler -f <filename>`
    1. Windows: `go build -o ./bin/compiler.exe ./cmd/compiler`
        1. Then: `.\bin\compiler.exe -f <filename>`

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
