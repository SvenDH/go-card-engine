extends Sprite3D

@onready var name_label: Label3D = $Name
@onready var cost_label: Label3D = $Costs
@onready var text_label: Label3D = $Text
@onready var stats_label: Label3D = $Stats
@onready var image_sprite: Sprite3D = $Image

func set_card_data(card_name: String, image: Texture2D, card_text: String, cost_text: String, stats_text: String, card_color: Color = Color.WHITE) -> void:
	name_label.text = card_name
	cost_label.text = cost_text
	text_label.text = card_text
	stats_label.text = stats_text

	if image:
		image_sprite.texture = image

	modulate = card_color
