package godot

import (
	"graphics.gd/classdb/BoxMesh"
	"graphics.gd/classdb/MeshInstance3D"
	"graphics.gd/classdb/Node3D"
	"graphics.gd/classdb/StandardMaterial3D"
	"graphics.gd/variant/Angle"
	"graphics.gd/variant/Color"
	"graphics.gd/variant/Euler"
	"graphics.gd/variant/Float"
	"graphics.gd/variant/Vector3"
)

// board3DScene manages 3D board slots and positioning.
type board3DScene struct {
	root            Node3D.Instance
	slotsPlayer     []MeshInstance3D.Instance
	slotsEnemy      []MeshInstance3D.Instance
	occupiedPlayer  map[int]*cardView
	occupiedEnemy   map[int]*cardView
	positionsPlayer []Vector3.XYZ
	positionsEnemy  []Vector3.XYZ
	cardHeight      Float.X
}

func newBoard3DScene(root Node3D.Instance, slotCount int, cardHeight Float.X) *board3DScene {
	scene := &board3DScene{
		root:            root,
		slotsPlayer:     make([]MeshInstance3D.Instance, 0, slotCount),
		slotsEnemy:      make([]MeshInstance3D.Instance, 0, slotCount),
		occupiedPlayer:  make(map[int]*cardView),
		occupiedEnemy:   make(map[int]*cardView),
		positionsPlayer: make([]Vector3.XYZ, 0, slotCount),
		positionsEnemy:  make([]Vector3.XYZ, 0, slotCount),
		cardHeight:      cardHeight,
	}

	ground := MeshInstance3D.New()
	groundMesh := BoxMesh.New()
	groundMesh.SetSize(Vector3.XYZ{8, 0.05, 6})
	ground.SetMesh(groundMesh.AsMesh())
	mat := StandardMaterial3D.New().AsBaseMaterial3D()
	mat.SetAlbedoColor(Color.RGBA{R: 0.16, G: 0.18, B: 0.2, A: 1})
	ground.SetSurfaceOverrideMaterial(0, mat.AsMaterial())
	ground.AsNode3D().SetPosition(Vector3.XYZ{0, -0.03, -2.0})
	root.AsNode().AddChild(ground.AsNode())

	spacing := 1.5
	startX := -spacing * float64(slotCount-1) / 2
	playerZ := -1.4
	enemyZ := -3.0

	playerMat := StandardMaterial3D.New().AsBaseMaterial3D()
	playerMat.SetAlbedoColor(Color.RGBA{R: 0.2, G: 0.25, B: 0.3, A: 1})
	enemyMat := StandardMaterial3D.New().AsBaseMaterial3D()
	enemyMat.SetAlbedoColor(Color.RGBA{R: 0.16, G: 0.2, B: 0.24, A: 1})

	for i := 0; i < slotCount; i++ {
		posX := Float.X(startX + float64(i)*spacing)

		playerSlot := MeshInstance3D.New()
		playerBox := BoxMesh.New()
		playerBox.SetSize(Vector3.XYZ{1.2, 0.02, 1.7})
		playerSlot.SetMesh(playerBox.AsMesh())
		playerSlot.SetSurfaceOverrideMaterial(0, playerMat.AsMaterial())
		playerPos := Vector3.XYZ{posX, 0, Float.X(playerZ)}
		playerSlot.AsNode3D().SetPosition(playerPos)
		root.AsNode().AddChild(playerSlot.AsNode())

		enemySlot := MeshInstance3D.New()
		enemyBox := BoxMesh.New()
		enemyBox.SetSize(Vector3.XYZ{1.2, 0.02, 1.7})
		enemySlot.SetMesh(enemyBox.AsMesh())
		enemySlot.SetSurfaceOverrideMaterial(0, enemyMat.AsMaterial())
		enemyPos := Vector3.XYZ{posX, 0, Float.X(enemyZ)}
		enemySlot.AsNode3D().SetPosition(enemyPos)
		root.AsNode().AddChild(enemySlot.AsNode())

		scene.slotsPlayer = append(scene.slotsPlayer, playerSlot)
		scene.slotsEnemy = append(scene.slotsEnemy, enemySlot)
		scene.positionsPlayer = append(scene.positionsPlayer, playerPos)
		scene.positionsEnemy = append(scene.positionsEnemy, enemyPos)
	}
	return scene
}

func (b *board3DScene) clear(view *cardView) {
	if b == nil || view == nil {
		return
	}
	for idx, v := range b.occupiedPlayer {
		if v == view {
			delete(b.occupiedPlayer, idx)
		}
	}
	for idx, v := range b.occupiedEnemy {
		if v == view {
			delete(b.occupiedEnemy, idx)
		}
	}
}

func (b *board3DScene) SlotCount() int {
	if b == nil {
		return 0
	}
	return len(b.positionsPlayer)
}

func (b *board3DScene) PlaceAt(view *cardView, index int, enemy bool) bool {
	if b == nil || view == nil || view.mesh == MeshInstance3D.Nil {
		return false
	}
	b.clear(view)

	positions := b.positionsPlayer
	occupied := b.occupiedPlayer
	if enemy {
		positions = b.positionsEnemy
		occupied = b.occupiedEnemy
	}
	if index < 0 || index >= len(positions) {
		return false
	}
	if existing, ok := occupied[index]; ok && existing != view {
		return false
	}

	target := positions[index]
	target.Y = 0.01

	view.mesh.AsNode3D().SetPosition(target)
	view.mesh.AsNode3D().SetRotation(Euler.Radians{X: Angle.Pi / 2, Y: 0, Z: 0})
	view.location = "board"
	view.fieldIndex = index
	occupied[index] = view
	return true
}

func (b *board3DScene) Remove(view *cardView) {
	if b == nil {
		return
	}
	b.clear(view)
}
