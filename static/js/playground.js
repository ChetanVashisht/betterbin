class Playground {
    constructor() {
        this.supportedLanguages = ['go', 'python', 'ruby'];
        this.initElements();
        this.setupEventListeners();
        this.hideCliContainer(); // Initially hide
        
        // Make clearViewer available globally
        window.clearViewer = () => {
            const pasteContent = document.getElementById('paste-content');
            if (pasteContent) {
                pasteContent.innerHTML = `
                    <div class="text-gray-500 text-center mt-4">
                        Select a paste to view its contents
                    </div>
                `;
            }
            
            const pasteList = document.getElementById('paste-list');
            if (pasteList) {
                pasteList.scrollTop = 0;
            }
            
            this.hideCliContainer();
        };
    }

    initElements() {
        this.terminal = document.getElementById('terminal');
        this.output = document.getElementById('output');
        this.input = document.getElementById('cli-input');
        this.runBtn = document.getElementById('run-btn');
        this.clearBtn = document.getElementById('clear-btn');
        this.cliContainer = document.getElementById('cli-container');
    }

    setupEventListeners() {
        if (this.runBtn) {
            this.runBtn.addEventListener('click', () => this.runCode());
        }
        if (this.clearBtn) {
            this.clearBtn.addEventListener('click', () => this.clear());
        }
        if (this.input) {
            this.input.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.handleInput(e.target.value);
                    e.target.value = '';
                }
            });
        }

        // Listen for HTMX content updates
        document.body.addEventListener('htmx:afterSettle', () => {
            this.checkLanguageSupport();
        });
    }

    hideCliContainer() {
        if (this.cliContainer) {
            this.cliContainer.style.display = 'none';
        }
    }

    checkLanguageSupport() {
        const codeElement = document.querySelector('#paste-content code');
        if (!codeElement || !this.cliContainer) {
            this.hideCliContainer();
            return;
        }

        const classes = codeElement.className.split(' ');
        const languageClass = classes.find(cls => cls.startsWith('language-'))?.replace('language-', '') || 'plaintext';
        
        this.cliContainer.style.display = this.supportedLanguages.includes(languageClass) ? 'block' : 'none';
    }

    async runCode() {
        const codeElement = document.querySelector('#paste-content code');
        const code = codeElement.textContent;
        const classes = codeElement.className.split(' ');
        const languageClass = classes.find(cls => cls.startsWith('language-'))?.replace('language-', '') || 'plaintext';

        // Map highlight.js languages to our supported ones

        const languageMap = {
            'go': 'go',
            'python': 'python',
            'ruby': 'ruby'
        };

        const language = languageMap[languageClass] || 'plaintext';

        try {
            const response = await fetch('/run', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ code, language })
            });

            const result = await response.json();
            this.appendOutput(result.output);
        } catch (error) {
            this.appendOutput(`Error: ${error.message}`, 'error');
        }
    }

    appendOutput(text, type = 'normal') {
        const line = document.createElement('div');
        line.textContent = text;
        if (type === 'error') {
            line.classList.add('text-red-500');
        }
        this.output.appendChild(line);
        this.terminal.scrollTop = this.terminal.scrollHeight;
    }

    handleInput(text) {
        // Send input to running program
        fetch('/input', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input: text })
        });
        this.appendOutput(`> ${text}`);
    }

    clear() {
        this.output.innerHTML = '';
    }
}

// Initialize the playground when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.playground = new Playground();
}); 