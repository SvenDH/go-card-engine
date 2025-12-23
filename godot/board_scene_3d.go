package godot

import (
	"math"

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
	root       Node3D.Instance
	slots      []MeshInstance3D.Instance
	occupied   map[int]*cardView
	positions  []Vector3.XYZ
	cardHeight Float.X
}

func newBoard3DScene(root Node3D.Instance, slotCount int, cardHeight Float.X) *board3DScene {
	scene := &board3DScene{
		root:       root,
		slots:      make([]MeshInstance3D.Instance, 0, slotCount),
		occupied:   make(map[int]*cardView),
		cardHeight: cardHeight,
	}

	ground := MeshInstance3D.New()
	groundMesh := BoxMesh.New()
	groundMesh.SetSize(Vector3.XYZ{8, 0.05, 6})
	ground.SetMesh(groundMesh.AsMesh())
	mat := StandardMaterial3D.New().AsBaseMaterial3D()
	mat.SetAlbedoColor(Color.RGBA{R: 0.16, G: 0.18, B: 0.2, A: 1})
	ground.SetSurfaceOverrideMaterial(0, mat.AsMaterial())
	ground.AsNode3D().SetPosition(Vector3.XYZ{0, -0.03, -3})
	root.AsNode().AddChild(ground.AsNode())

	spacing := 1.6
	startX := -spacing * float64(slotCount-1) / 2
	playerZ := -2.0

	for i := 0; i < slotCount; i++ {
		slot := MeshInstance3D.New()
		box := BoxMesh.New()
		box.SetSize(Vector3.XYZ{1.2, 0.02, 1.7})
		slot.SetMesh(box.AsMesh())
		m := StandardMaterial3D.New().AsBaseMaterial3D()
		m.SetAlbedoColor(Color.RGBA{R: 0.2, G: 0.25, B: 0.3, A: 1})
		slot.SetSurfaceOverrideMaterial(0, m.AsMaterial())

		pos := Vector3.XYZ{Float.X(startX + float64(i)*spacing), 0, Float.X(playerZ)}
		slot.AsNode3D().SetPosition(pos)
		root.AsNode().AddChild(slot.AsNode())

		scene.slots = append(scene.slots, slot)
		scene.positions = append(scene.positions, pos)
	}
	return scene
}

func (b *board3DScene) clear(view *cardView) {
	for idx, v := range b.occupied {
		if v == view {
			delete(b.occupied, idx)
			return
		}
	}
}

func (b *board3DScene) nearestSlot(pos Vector3.XYZ) (int, float64) {
	best := -1
	bestDist := math.MaxFloat64
	for i, p := range b.positions {
		dx := float64(pos.X - p.X)
		dz := float64(pos.Z - p.Z)
		dist := math.Hypot(dx, dz)
		if dist < bestDist {
			bestDist = dist
			best = i
		}
	}
	return best, bestDist
}

func (b *board3DScene) Place(view *cardView, pos Vector3.XYZ) bool {
	if b == nil || view == nil || view.mesh == MeshInstance3D.Nil {
		return false
	}
	b.clear(view)

	idx, dist := b.nearestSlot(pos)
	if idx < 0 || dist > 1.4 {
		return false
	}
	if existing, ok := b.occupied[idx]; ok && existing != view {
		return false
	}

	target := b.positions[idx]
	// Lay the card flat on the board surface.
	target.Y = 0.01

	view.mesh.AsNode3D().SetPosition(target)
	view.mesh.AsNode3D().SetRotation(Euler.Radians{X: Angle.Pi / 2, Y: 0, Z: 0})
	view.location = "board"
	view.fieldIndex = idx
	b.occupied[idx] = view
	return true
}

func (b *board3DScene) Remove(view *cardView) {
	if b == nil {
		return
	}
	b.clear(view)
}
