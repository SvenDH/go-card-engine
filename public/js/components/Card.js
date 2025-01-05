const COLOR_MAIN = 0x4e342e;
const COLOR_LIGHT = 0x7b5e57;
const COLOR_DARK = 0x260e04;

export default class Card extends Phaser.GameObjects.Container {
  	constructor({ scene, x, y, id, name, image, location = 'deck', faceup = false }) {
		let spriteCard = new Phaser.GameObjects.Sprite(scene, 0, 0, faceup ? 'card' : 'back').setOrigin(0.5, 1).setScale(2.0);
		let spriteImage = new Phaser.GameObjects.Sprite(scene, 0, -68, image).setOrigin(0.5, 1).setScale(2.0);
		let textName = new Phaser.GameObjects.BitmapText(scene, -66, -154, 'pressstart', name, 12, Phaser.GameObjects.BitmapText.ALIGN_CENTER);
		super(scene, x, y, [spriteCard, spriteImage, textName]);
		this.spriteCard = spriteCard;
		this.spriteImage = spriteImage;
		this.textName = textName;
		this.abilities = [];
		this.abilityMenu = null;
		this.cardname = name;
		this.scene = scene;
		this.id = id;
		this.location = location;
		this.index = -1;
		this.isDragging = false;

		this.scene.add.existing(this);

		this.setActive(false);
		this.setFaceup(faceup);

		this.on("pointerdown", () => {
			if (scene.player.board.contains(this)) {
				scene.conn.send({ type: 'game.choice', data: [this.id] });
			}
		}, this);
    
		this.on("dragstart", (pointer) => {
			if (scene.player.hand.contains(this)) {
				scene.player.hand.removeCard(this);
				this.isDragging = true;

				this.setDepth(scene.player.hand.countActive());

				scene.tweens.add({
					targets: this,
					angle: 0,
					x: pointer.x,
					y: pointer.y,
					scale: 2.0,
					duration: 150
				});
				
				scene.tweens.add({
					targets: scene.background,
					alpha: 0.3,
					duration: 150
				});
				
				scene.conn.send({
					type: 'game.choice',
					data: [this.id],
				});
			};
		}, this);
		
		this.on("drag", (pointer) => {
			if (!scene.player.hand.contains(this) && !scene.player.board.contains(this)) {
				this.x = pointer.x;
				this.y = pointer.y;
			}
		}, this);

		this.on("drop", (pointer, target) => {
			for (const player of Object.values(scene.players)) {
				if (player.board.zones.includes(target)) {
					let index = player.board.zones.indexOf(target);
					player.board.addCard(index, this);
					scene.conn.send({ type: 'game.choice', data: [index.toString()] });
					return;
				}
			}
		}, this);

		this.on("dragend", (pointer, dragX, dragY, dropped) => {
			if (!scene.player.hand.contains(this) && !scene.player.board.contains(this)) {
				this.isDragging = false;

				if(!dropped) {
					scene.player.hand.addCard(this);
					scene.conn.send({ type: 'game.choice', data: [] });
				}

				scene.tweens.add({
					targets: scene.background,
					alpha: 1,
					duration: 150
				});

				this.setActive(this.isActive);
			}
		}, this);
    }

    setCardName(value) {
		this.textName.setText(value);
		this.cardname = value;
	}

	setAbilities(abilities) {
		this.abilities = abilities;
	}

	setActive(isActive) {
		this.isActive = isActive;
		if (this.isDragging) return;

		if (isActive) {
			this.setInteractive({
				hitArea: new Phaser.Geom.Rectangle(-75, -164, 75, 164),
				hitAreaCallback: Phaser.Geom.Rectangle.Contains,
				draggable: isActive
			});
		} else {
			this.disableInteractive();
		}
		this.spriteCard.setTint(isActive ? 0xffffff : 0x7f7f7f);
		this.spriteImage.setTint(isActive ? 0xffffff : 0x7f7f7f);
		this.textName.setTint(isActive ? 0xffffff : 0x7f7f7f);
	}

  	setFaceup(isFaceUp) {
		if (isFaceUp === this.isFaceUp) return;
		this.scene.tweens.addMultiple([
			{
				targets: this.spriteCard,
				scaleX: 0,
				duration: 100,
				yoyo: true,
				onYoyo: () => this.spriteCard.setTexture(isFaceUp ? 'card' : 'back')
			},
			{
				targets: this.spriteImage,
				scaleX: 0,
				duration: 100,
				yoyo: true,
				onYoyo: () => this.spriteImage.visible = isFaceUp
			},
			{
				targets: this.textName,
				scaleX: 0,
				duration: 100,
				yoyo: true,
				onYoyo: () => this.textName.visible = isFaceUp
			}
		]);
		this.isFaceUp = isFaceUp;
	}

	openAbilityMenu(options) {
		let scene = this.scene;
		let abilities = options.map((option) => ({ text: this.abilities[option], children: [] }));
		this.abilityMenu = scene.rexUI.add.menu({
			items: abilities,
			createBackgroundCallback: function (items) {
				return scene.rexUI.add.roundRectangle(0, 0, 2, 2, 0, COLOR_MAIN);
			},
			createButtonCallback: function (item, i, items) {
				return scene.rexUI.add.label({
					background: scene.rexUI.add.roundRectangle(0, 0, 2, 2, 0),
					text: scene.add.rexBBCodeText(0, 0, item.text, {fontFamily: 'Arial', fontSize: '8px'}),
					//icon: scene.rexUI.add.roundRectangle(0, 0, 0, 0, 4, COLOR_DARK),
					space: {
						left: 2,
						right: 2,
						top: 1,
						bottom: 1,
						icon: 1
					}
				})
			},
		});
		this.add(this.abilityMenu);
		//this.abilityMenu.collapse();
		
	}
}
