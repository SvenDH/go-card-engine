import Card from '../components/Card.js';
import Board from '../components/Board.js';
import Hand from '../components/Hand.js';

export default class Game extends Phaser.Scene {
	constructor() {
		super('Game');
	}

	init(data) {
		this.conn = data.conn;
		this.room = data.room;
		this.name = data.name;
	}

	preload() {
		this.load.setPath('assets');
		this.load.image('card', 'card.png');
		this.load.image('back', 'back.png');
		this.load.image('avatar', 'avatar.png');
		this.load.image('background', 'background.png');
		this.load.image('field', 'field.png');
    	this.load.bitmapFont('pressstart', 'pressstart.png', 'pressstart.fnt');
	}

	create() {
		this.player = null;
		this.players = {};
		this.cards = {};
		this.seen = {};
		this.cardInstances = {};

		this.background = this.add.sprite(this.game.config.width / 2, this.game.config.height / 2, "background");

		this.conn.on('game.info', (event) => {
			const message = event.data;
			for (const id in message.players || {}) {
				if (this.players[id] === undefined) this.players[id] = {};
				this.players[id].name = message.players[id];
				this.players[id].id = id;
				if (message.players[id] === this.name) {
					// This player
					this.player = this.players[id];
					this.players[id].hand = this.add.existing(new Hand({
						id,
						scene: this,
						x: this.game.config.width / 2,
						y: this.game.config.height,
					}));
					this.players[id].board = this.add.existing(new Board({
						id,
						scene: this,
						x: 160,
						y: 460,
					}));
				} else {
					// Other players
					this.players[id].hand = this.add.existing(new Hand({
						id,
						scene: this,
						x: this.game.config.width / 2,
						y: 0,
					}));
					this.players[id].board = this.add.existing(new Board({
						id,
						scene: this,
						x: 160,
						y: 100,
					}));
				}
			}
			for (let i = 0; i < message.cards?.length || 0; i++) {
				let card = message.cards[i].split('\n');
				let cardName = card[0].split(' {')[0];
				let cardCosts = card[0].split(' {')[1]?.split('}')[0].split('}{');
				let cardType = card[1];
				let powerHealth = card[card.length - 1].split(' / ');
				let cardText;
				if (powerHealth.length > 1) {
					cardText = card.slice(2, card.length - 2).join('\n');
				} else {
					cardText = card.slice(2, card.length - 1).join('\n');
					powerHealth = null;
				}
				this.cards[cardName] = {
					name: cardName,
					costs: cardCosts,
					type: cardType,
					text: cardText,
					power: powerHealth && powerHealth[0],
					health: powerHealth && powerHealth[1],
				};
			}
			for (const prop in message.seen || {}) {
				this.seen[prop] = this.cards[message.seen[prop]];
				if (this.cardInstances[prop]) {
					this.cardInstances[prop].setCardName(this.seen[prop].name);
					// TODO: avatar
				}
			}
		});

		this.conn.on('game.event', (event) => {
			const message = event.data;
			if (message.event === 'draw') {
				let card = new Card({
					id: message.subject,
					scene: this,
					x: this.game.config.width,
					y: this.game.config.height / 2,
					name: this.seen[message.subject]?.name,
					card: 'card',
					image: 'avatar'
				});
				this.cardInstances[message.subject] = card;
				this.players[message.controller].hand.addCard(card);
			} else if (message.event === 'enter-board') {
				let card = this.cardInstances[message.subject];
				this.players[message.controller].board.addCard(message.args[0], card);
			}
		});

		this.conn.on('game.prompt', (event) => {
			const message = event.data;
			if (message.action === 'card') {
				for (const card in this.cardInstances) {
					this.cardInstances[card].setActive(message.options.includes(card));
				}
				for (const zone in this.players[this.player.id].board.zones) {
					this.players[this.player.id].board.setActive(zone, false);
				}
			} else if (message.action === 'field') {
				for (const card in this.cardInstances) {
					this.cardInstances[card].setActive(false);
				}
				for (const zone in this.players[this.player.id].board.zones) {
					this.players[this.player.id].board.setActive(zone, message.options.includes(zone.toString()));
				}
			}
		});

		this.conn.send({
			type: 'game.ready',
			data: this.room,
		});
	}

	update() {}
}
