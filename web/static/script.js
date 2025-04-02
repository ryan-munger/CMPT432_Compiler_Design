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
        verboseButton.classList.toggle("bg-red-600", !verboseFlag);
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

    const testSelector = document.getElementById("exampleSelector");
    testSelector.addEventListener("change", function () {
        const selectedTest = this.value;
        if (selectedTest) {
            loadExampleContent(selectedTest);
        }
    });

    // Function to load test content
    function loadExampleContent(testName) {
        fetch(`/static/examples/${testName}.txt`)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Failed to load ${testName}.txt`);
                }
                return response.text();
            })
            .then(data => {
                codeInput.value = data;
                document.getElementById("consoleOutput").textContent = "";
            })
            .catch(error => {
                codeInput.textContent = "Error loading test: " + error.message;
                document.getElementById("consoleOutput").textContent = "";
            });
    }

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
                if (data.output) {
                    document.getElementById("consoleOutput").innerHTML = data.output;

                    // After successfully receiving the compilation output, send GET requests 
                    return Promise.all([
                        fetch("/getSymbolTables").then(response => response.json()),
                        fetch("/getMachineCode").then(response => response.text()),
                        fetch("/getCST").then(response => response.text()),
                        fetch("/getAST").then(response => response.text())
                    ]);
                } else {
                    throw new Error("Compilation failed");
                }
            })
            .then(([symbolData, machineCodeData, cstData, astData]) => {
                // for evil tailwind
                symbolData = symbolData.replace(/<table/g, '<table class="w-full border border-gray-500"');
                symbolData = symbolData.replace(/<th/g, '<th class="border border-gray-500 px-2 py-1 bg-gray-600 text-white"');
                symbolData = symbolData.replace(/<td/g, '<td class="border border-gray-500 px-2 py-1 text-green-400"');
                document.getElementById("symbolTable").innerHTML = symbolData;
                document.getElementById('machineCode').textContent = machineCodeData;
                document.getElementById('cstBox').textContent = cstData;
                document.getElementById('astBox').textContent = astData;
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

function copyMachineCode() {
    const machineCode = document.getElementById('machineCode');
    const text = machineCode.textContent;
    navigator.clipboard.writeText(text)
        .then(() => {
            // Show a brief "Copied!" message
            const copyBtn = document.getElementById('copyButton');
            const originalText = copyBtn.innerHTML;
            copyBtn.innerHTML = "Copied!";
            setTimeout(() => {
                copyBtn.innerHTML = originalText;
            }, 2000);
        })
        .catch(err => {
            console.error('Failed to copy: ', err);
        });
}