import { baseUrl } from '../config.js';
import Connection from '../components/Connection.js';


export default class Login extends Phaser.Scene {
    constructor() {
        super('Login');
    }

    preload() {
        this.load.plugin('rexinputtextplugin', 'https://raw.githubusercontent.com/rexrainbow/phaser3-rex-notes/master/dist/rexinputtextplugin.min.js', true);
    }

    create() {
        this.username = '';
        this.password = '';
        
        var inputStyle = {
            fontSize: '12px',
            color: '#ffffff',
            border: 1,
            backgroundColor: '#333333',
            borderColor: '#111111',
            borderRadius: 10,
            outline: 'none',
        };
        this.add.rexInputText(400, 200, 10, 10, {
            type: 'text',
            placeholder: 'Username',
            ...inputStyle
        })
        .resize(200, 40)
        .setOrigin(0.0)
        .on('textchange', (inputText) => {
            this.username = inputText.text;
        });
        
        this.add.rexInputText(400, 240, 10, 10, {
            type: 'password',
            placeholder: 'Password',
            ...inputStyle
        })
        .resize(200, 40)
        .setOrigin(0.0)
        .on('textchange', (inputText) => {
            this.password = inputText.text;
        });

        this.loginButton = this.add.text(400, 300, ['Login'])
        .setFontSize(18)
        .setFontFamily('Trebuchet MS')
        .setColor('#00ffff')
        .setInteractive();
        this.loginButton.on('pointerdown', () =>
            fetch(`${baseUrl}/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username: this.username, password: this.password })
            })
            .then(res => res.json())
            .then(data => {
                localStorage.setItem('token', JSON.stringify(data));
                this.open(data);
            })
        );
        
        var existing = localStorage.getItem('token');
        if (existing) {
            this.open(JSON.parse(existing));
        }
    }

    open(token) {
        var conn = new Connection(token.access_token, () => {
            console.log('Connected!');
            this.scene.start('Lobby', { conn, name: token.name });
        });
    }

    update() {}
}
