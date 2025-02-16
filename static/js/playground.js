class Playground {
    constructor() {
        this.terminal = document.getElementById('terminal');
        this.output = document.getElementById('output');
        this.input = document.getElementById('cli-input');
        this.runBtn = document.getElementById('run-btn');
        this.clearBtn = document.getElementById('clear-btn');
        
        this.setupEventListeners();
    }

    setupEventListeners() {
        this.runBtn.addEventListener('click', () => this.runCode());
        this.clearBtn.addEventListener('click', () => this.clear());
        this.input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.handleInput(e.target.value);
                e.target.value = '';
            }
        });
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
            'javascript': 'javascript',
            'java': 'java',
            'cpp': 'cpp',
            'csharp': 'csharp',
            'php': 'php',
            'ruby': 'ruby',
            'swift': 'swift',
            'rust': 'rust',
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

new Playground(); 