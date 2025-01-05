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

		this.load.scenePlugin({
			key: 'rexuiplugin',
			url: '../js/plugins/rexuiplugin.min.js',
			sceneKey: 'rexUI'
		});
		this.load.plugin(
			'rexbbcodetextplugin',
			'../js/plugins/rexbbcodetextplugin.min.js',
			true
		);
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
				let card = message.cards[i];
				this.cards[card.name] = {
					name: card.name,
					costs: card.costs,
					type: card.types,
					subtypes: card.subtypes,
					keywords: card.keywords,
					activated: card.activated,
					triggered: card.triggered,
					static: card.static,
					power: card.power,
					health: card.health,
				};
			}
			for (const prop in message.seen || {}) {
				this.seen[prop] = this.cards[message.seen[prop]];
				if (this.cardInstances[prop]) {
					this.cardInstances[prop].setCardName(this.seen[prop].name);
					this.cardInstances[prop].setAbilities(this.seen[prop].activated);
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
			this.disableAll();
			if (message.action === 'card') {
				for (const card in this.cardInstances) {
					this.cardInstances[card].setActive(message.options.includes(card));
				}
			} else if (message.action === 'field') {
				for (const zone in this.players[this.player.id].board.zones) {
					this.players[this.player.id].board.setActive(zone, message.options.includes(zone.toString()));
				}
			} else if (message.action === 'ability') {
				this.cardInstances[message.card].openAbilityMenu(message.options.map((o) => parseInt(o)));
			}
		});

		this.conn.send({ type: 'game.ready', data: this.room });
	}

	update() {}

	disableAll() {
		for (const card in this.cardInstances) {
			this.cardInstances[card].setActive(false);
		}
		for (const playerId in this.players) {
			const player = this.players[playerId];
			for (const zone in player.board.zones) {
				player.board.setActive(zone, false);
			}
		}
	}
}
