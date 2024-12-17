export default class Hand extends Phaser.GameObjects.Group {
	constructor({ id, scene, x, y }) {
		super(scene, []);
		this.scene = scene;
		this.id = id;
        this.x = x;
        this.y = y;
    }

	addCard(card) {
		this.add(card);
		card.location = 'hand';
        card.setFaceup(true);

		this.rearangeCards();
	}

	removeCard(card) {
		this.remove(card);
		this.rearangeCards();
	}

	rearangeCards() {
		this.children.iterate((card, i) => {
			card.setDepth(i);
			let totalCards = this.countActive();
			let rotation = -Math.PI / 4 / totalCards * ((totalCards / 2) - i);
			let x = this.x + 200 * Math.cos(-rotation + Math.PI / 2);
			let y = this.y + 200 - 200 * Math.sin(-rotation + Math.PI / 2);
			this.scene.tweens.add({
				targets: card,
				rotation,
				x,
				y,
				scale: 2.0,
				duration: 150,
			});
		}, this);
	}
}