export default class Lobby extends Phaser.Scene {
    constructor() {
        super('Lobby');
    }

    init(data) {
        this.conn = data.conn;
        this.name = data.name;
    }

    preload() {
        this.load.plugin('rexinputtextplugin', 'https://raw.githubusercontent.com/rexrainbow/phaser3-rex-notes/master/dist/rexinputtextplugin.min.js', true);
    }

    create() {
        this.conn.on('room.joined', (message) =>
            this.scene.start('Game', {
                conn: this.conn,
                room: message.target,
                name: this.name,
            })
        );

        this.username = '';
        
        this.add.rexInputText(400, 240, 10, 10, {
            type: 'text',
            placeholder: 'Username',
            fontSize: '12px',
            color: '#ffffff',
            border: 1,
            backgroundColor: '#333333',
            borderColor: '#111111',
            borderRadius: 10,
            outline: 'none',
        })
        .resize(200, 40)
        .setOrigin(0.0)
        .on('textchange', (inputText) => {
            this.username = inputText.text;
        });

        this.inviteButton = this.add.text(400, 300, ['Invite'])
        .setFontSize(18)
        .setFontFamily('Trebuchet MS')
        .setColor('#00ffff')
        .setInteractive();
        this.inviteButton.on('pointerdown', () =>
            this.conn.send({
                type: 'room.join-private',
                data: this.username
            })
        );

        this.campaignButton = this.add.text(400, 400, ['Campaign'])
        .setFontSize(18)
        .setFontFamily('Trebuchet MS')
        .setColor('#00ffff')
        .setInteractive();
        this.campaignButton.on('pointerdown', () =>
            this.conn.send({
                type: 'room.join-npc',
                data: 'player2'
            })
        );
    }

    update() {}
}
