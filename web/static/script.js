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
        verboseButton.classList.toggle("bg-red-500", !verboseFlag);
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
                updateLineNumbers();
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

    const lineNumbers = document.getElementById('lineNumbers');
    const clearButton = document.getElementById('clearButton');

    // Function to update line numbers
    function updateLineNumbers() {
        const lines = codeInput.value.split('\n');
        let lineNumbersText = '';

        for (let i = 1; i <= lines.length; i++) {
            lineNumbersText += i + '\n';
        }

        lineNumbers.textContent = lineNumbersText;
    }

    // Initialize with at least one line number
    lineNumbers.textContent = '1';

    // Add event listeners
    codeInput.addEventListener('input', updateLineNumbers);
    codeInput.addEventListener('scroll', () => {
        lineNumbers.scrollTop = codeInput.scrollTop;
    });

    // Match line height between textarea and line numbers
    codeInput.addEventListener('keydown', (e) => {
        if (e.key === 'Tab') {
            e.preventDefault();
            const start = codeInput.selectionStart;
            const end = codeInput.selectionEnd;

            codeInput.value = codeInput.value.substring(0, start) + '    ' + codeInput.value.substring(end);
            codeInput.selectionStart = codeInput.selectionEnd = start + 4;
            updateLineNumbers();
        }
    });

    // Clear button functionality
    clearButton.addEventListener('click', () => {
        codeInput.value = '';
        updateLineNumbers();
    });

    // Ensure line numbers are updated when the page loads
    window.addEventListener('load', () => {
        updateLineNumbers();
        // Set focus to the textarea
        codeInput.focus();
    });


    const dropdownButton = document.getElementById('exampleSelector');
    const dropdownContent = document.getElementById('dropdownContent');

    // Toggle dropdown when clicking the button
    dropdownButton.addEventListener('click', function () {
        dropdownContent.classList.toggle('hidden');
    });

    // Handle click events for all menu items with data-value
    document.querySelectorAll('[data-value]').forEach(item => {
        item.addEventListener('click', function (e) {
            e.preventDefault();
            const value = this.getAttribute('data-value');
            dropdownButton.textContent = this.textContent;
            dropdownContent.classList.add('hidden');

            // Dispatch a custom event that your existing code can listen for
            const event = new CustomEvent('exampleSelected', {
                detail: { value: value }
            });
            document.dispatchEvent(event);
        });
    });

    // Close the dropdown when clicking outside
    window.addEventListener('click', function (e) {
        if (!dropdownButton.contains(e.target) && !dropdownContent.contains(e.target)) {
            dropdownContent.classList.add('hidden');
        }
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