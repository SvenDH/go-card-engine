

export default class Board extends Phaser.GameObjects.Group {
	constructor({ id, scene, x, y, columns = 5, rows = 2, fieldWidth = 150, fieldHeight = 164, marginX = 5, marginY = 5 }) {
		super(scene, []);
		this.columns = columns;
		this.rows = rows;
		this.scene = scene;
		this.id = id;
        this.zones = [];
        this.cards = [];
		this.sprites = [];
		this.active = [];
        for (let row = 0; row < rows; row++) {
            for (let column = 0; column < columns; column++) {
                var zx = x + column * (fieldWidth + marginX);
                var zy = y + row * (fieldHeight + marginY);
		        this.sprites.push(scene.add.sprite(zx, zy, 'field').setScale(2.0));
                this.zones.push(
                    scene.add.zone(zx, zy, fieldWidth, fieldHeight)
                    .setRectangleDropZone(fieldWidth, fieldHeight)
					.disableInteractive());
                this.cards.push(null);
				this.active.push(false);
            }
        }
    }

	addCard(index, card) {
		this.cards[index] = card;
		this.add(card);
		card.index = index;
		card.location = 'field';
		card.setDepth(0);

		let zone = this.zones[index];
		this.scene.tweens.add({
			targets: card,
			angle: 0,
			x: zone.x,
			y: zone.y + 82,
			scale: 2.0,
			duration: 150,
			callbackScope: card,
			onComplete: () => {
				this.scene.cameras.main.shake(300, 0.02);
			}
		});
	}

	setActive(index, isActive) {
		if (isActive) {
			this.zones[index].setInteractive();
			this.sprites[index].setTint(0xffffff);
		} else {
			this.zones[index].disableInteractive();
			this.sprites[index].setTint(0x7f7f7f);
		}
		this.active[index] = isActive;
	}
}