import Game from './scenes/Game.js';
import Lobby from './scenes/Lobby.js';
import Login from './scenes/Login.js';

//  Find out more information about the Game Config at:
//  https://newdocs.phaser.io/docs/3.70.0/Phaser.Types.Core.GameConfig
const config = {
    type: Phaser.AUTO,
    width: 1024,
    height: 768,
    parent: 'game-container',
    backgroundColor: '#111111',
    dom: {
        createContainer: true
    },
    input: {
        mouse: {
            target: 'game-container'
        },
        touch: {
            target: 'game-container'
        },
    },
    scale: {
        mode: Phaser.Scale.FIT,
        autoCenter: Phaser.Scale.CENTER_BOTH
    },
    scene: [
        Login,
        Lobby,
        Game,
    ]
};

export default new Phaser.Game(config);
