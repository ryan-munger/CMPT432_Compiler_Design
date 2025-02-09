document.addEventListener("DOMContentLoaded", function () {
    let verboseFlag = true;

    // Set initial button state
    const verboseButton = document.getElementById("verboseButton");
    verboseButton.classList.add("bg-green-500");
    verboseButton.textContent = "Verbose Mode: ON";

    // Toggle verbose mode
    verboseButton.addEventListener("click", function () {
        verboseFlag = !verboseFlag;
        verboseButton.textContent = "Verbose Mode: " + (verboseFlag ? "ON" : "OFF");
        verboseButton.classList.toggle("bg-green-500", verboseFlag);
        verboseButton.classList.toggle("bg-gray-700", !verboseFlag);
    });

    // Capture Tab Key in the textarea
    const codeInput = document.getElementById("codeInput");
    codeInput.addEventListener("keydown", function (e) {
        if (e.key === "Tab") {
            e.preventDefault();
            let start = this.selectionStart;
            let end = this.selectionEnd;

            // Insert four spaces for indentation
            this.value = this.value.substring(0, start) + "    " + this.value.substring(end);
            this.selectionStart = this.selectionEnd = start + 4;
        }
    });

    // Compile code function
    document.getElementById("compileButton").addEventListener("click", function () {
        const code = document.getElementById("codeInput").value;
        const verbose = document.getElementById("verboseButton").textContent.includes("ON");
    
        fetch("/compile", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({ code: code, verbose: verbose }),
        })
        .then(response => response.json())
        .then(data => {
            document.getElementById("consoleOutput").innerHTML = data.output;
        })
        .catch(error => {
            document.getElementById("consoleOutput").textContent = "Error: " + error;
        });
    });

    // Clear code function
    document.getElementById("clearButton").addEventListener("click", function () {
        codeInput.value = "";
        document.getElementById("consoleOutput").textContent = "";
    });
});
