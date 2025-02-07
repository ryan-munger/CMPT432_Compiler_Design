document.addEventListener("DOMContentLoaded", function () {
    let flag = true;

    // Set initial button state
    const flagButton = document.getElementById("flagButton");
    flagButton.classList.add("bg-green-500");
    flagButton.textContent = "Verbose Mode: ON";

    // Toggle verbose mode
    flagButton.addEventListener("click", function () {
        flag = !flag;
        flagButton.textContent = "Verbose Mode: " + (flag ? "ON" : "OFF");
        flagButton.classList.toggle("bg-green-500", flag);
        flagButton.classList.toggle("bg-gray-700", !flag);
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
        const verbose = document.getElementById("flagButton").textContent.includes("ON");
    
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
