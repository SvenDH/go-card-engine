export default class Card extends Phaser.GameObjects.Container {
  	constructor({ scene, x, y, id, name, image, location = 'deck', faceup = false }) {
		let spriteCard = new Phaser.GameObjects.Sprite(scene, 0, 0, faceup ? 'card' : 'back').setOrigin(0.5, 1);
		let spriteImage = new Phaser.GameObjects.Sprite(scene, 0, -34, image).setOrigin(0.5, 1);
		let textName = new Phaser.GameObjects.BitmapText(scene, -33, -77, 'pressstart', name, 8, Phaser.GameObjects.BitmapText.ALIGN_CENTER);
		super(scene, x, y, [spriteCard, spriteImage, textName]);
		this.spriteCard = spriteCard;
		this.spriteImage = spriteImage;
		this.textName = textName;
		this.cardname = name;
		this.scene = scene;
		this.id = id;
		this.location = location;
		this.index = -1;
		this.isDragging = false;
		this.scene.add.existing(this);
		this.setActive(false);
		this.setFaceup(faceup);
    
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
					scale: 4.0,
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
					scene.conn.send({
						type: 'game.choice',
						data: [index.toString()],
					});
					return;
				}
			}
			scene.player.hand.addCard(this);
		}, this);

		this.on("dragend", (pointer, dragX, dragY, dropped) => {
			if (!scene.player.hand.contains(this) && !scene.player.board.contains(this)) {
				this.isDragging = false;

				if(!dropped) {
					scene.player.hand.addCard(this);
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

	setActive(isActive) {
		this.isActive = isActive;
		if (this.isDragging) return;

		if (isActive) {
			this.setInteractive({
				hitArea: new Phaser.Geom.Rectangle(-37.5, -82, 75, 82),
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
}
