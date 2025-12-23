package godot

import (
	"math"

	"graphics.gd/classdb/BoxMesh"
	"graphics.gd/classdb/Camera3D"
	"graphics.gd/classdb/DirectionalLight3D"
	"graphics.gd/classdb/MeshInstance3D"
	"graphics.gd/classdb/Node"
	"graphics.gd/classdb/Node3D"
	"graphics.gd/classdb/Resource"
	"graphics.gd/classdb/StandardMaterial3D"
	"graphics.gd/classdb/SubViewport"
	"graphics.gd/classdb/Texture2D"
	"graphics.gd/variant/Angle"
	"graphics.gd/variant/Color"
	"graphics.gd/variant/Euler"
	"graphics.gd/variant/Float"
	"graphics.gd/variant/Vector2"
	"graphics.gd/variant/Vector3"
)

// hand3DScene renders hand cards as 3D meshes inside a SubViewport.
type hand3DScene struct {
	ui          *CardGameUI
	viewport    SubViewport.Instance
	root        Node3D.Instance
	camera      Camera3D.Instance
	light       DirectionalLight3D.Instance
	cardTexture Texture2D.Instance
	cards       map[*cardView]MeshInstance3D.Instance

	cardSpacing float64
	cardTilt    float64
	cardLift    float64
	baseDepth   float64
	cardSize    Vector3.XYZ
	cameraYaw   float64
}

func newHand3DScene(ui *CardGameUI, viewport SubViewport.Instance) *hand3DScene {
	if viewport == SubViewport.Nil {
		return nil
	}

	root := Node3D.New()
	viewport.AsNode().AddChild(root.AsNode())

	camera := Camera3D.New()
	camera.AsNode3D().SetPosition(Vector3.XYZ{0, 2.4, 6})
	root.AsNode().AddChild(camera.AsNode())
	camera.AsNode3D().LookAt(Vector3.XYZ{0, 0.1, 0})
	yaw := camera.AsNode3D().GlobalRotationDegrees().Y

	light := DirectionalLight3D.New()
	light.AsNode3D().SetRotationDegrees(Euler.Degrees{
		X: Angle.Degrees(-45),
		Y: Angle.Degrees(30),
		Z: Angle.Degrees(0),
	})
	root.AsNode().AddChild(light.AsNode())

	cardTexture := Resource.Load[Texture2D.Instance]("res://graphics/textures/card.png")

	return &hand3DScene{
		ui:          ui,
		viewport:    viewport,
		root:        root,
		camera:      camera,
		light:       light,
		cardTexture: cardTexture,
		cards:       make(map[*cardView]MeshInstance3D.Instance),
		cardSpacing: 1.35,
		cardTilt:    6,
		cardLift:    0.35,
		baseDepth:   3.0,
		cardSize:    Vector3.XYZ{1.2, 0.05, 1.7},
		cameraYaw:   float64(yaw),
	}
}

func (h *hand3DScene) Add(view *cardView) {
	if h == nil || view == nil {
		return
	}
	if _, ok := h.cards[view]; ok {
		return
	}

	mesh := MeshInstance3D.New()
	box := BoxMesh.New()
	box.SetSize(h.cardSize)
	mesh.SetMesh(box.AsMesh())

	material := StandardMaterial3D.New()
	baseMaterial := material.AsBaseMaterial3D()
	baseMaterial.SetAlbedoColor(Color.RGBA{R: 1, G: 1, B: 1, A: 1})
	if h.cardTexture != Texture2D.Nil {
		baseMaterial.SetAlbedoTexture(h.cardTexture)
	}
	mesh.SetSurfaceOverrideMaterial(0, material.AsMaterial())

	mesh.AsNode3D().SetPosition(Vector3.XYZ{0, 0, float32(-h.baseDepth)})
	h.root.AsNode().AddChild(mesh.AsNode())
	h.cards[view] = mesh
	view.mesh = mesh
}

func (h *hand3DScene) Remove(view *cardView) {
	if h == nil || view == nil {
		return
	}
	mesh, ok := h.cards[view]
	if !ok {
		return
	}
	delete(h.cards, view)
	view.mesh = MeshInstance3D.Nil
	if parent := mesh.AsNode().GetParent(); parent != Node.Nil {
		parent.RemoveChild(mesh.AsNode())
	}
}

func (h *hand3DScene) rootNode() Node3D.Instance {
	if h == nil {
		return Node3D.Nil
	}
	return h.root
}

// HoverAt tests a screen position against card meshes and marks the closest hit hovered.
func (h *hand3DScene) HoverAt(screen Vector2.XY) *cardView {
	if h == nil || h.ui.dragging != nil {
		return nil
	}
	origin := h.camera.ProjectRayOrigin(screen)
	dir := h.camera.ProjectRayNormal(screen)

	best := (*cardView)(nil)
	bestT := math.MaxFloat64
	for view, mesh := range h.cards {
		if view == nil || mesh == MeshInstance3D.Nil {
			continue
		}
		pos := mesh.AsNode3D().Position()
		if math.Abs(float64(dir.Y)) < 0.0001 {
			continue
		}
		t := (float64(pos.Y) - float64(origin.Y)) / float64(dir.Y)
		if t <= 0 || t >= bestT {
			continue
		}
		hitX := float64(origin.X) + float64(dir.X)*t
		hitZ := float64(origin.Z) + float64(dir.Z)*t
		dx := hitX - float64(pos.X)
		dz := hitZ - float64(pos.Z)
		yawDeg := float64(mesh.AsNode3D().RotationDegrees().Y)
		yaw := yawDeg * math.Pi / 180
		sin := math.Sin(-yaw)
		cos := math.Cos(-yaw)
		localX := dx*cos - dz*sin
		localZ := dx*sin + dz*cos
		if math.Abs(localX) <= float64(h.cardSize.X)/2 && math.Abs(localZ) <= float64(h.cardSize.Z)/2 {
			best = view
			bestT = t
		}
	}
	changed := false
	for view := range h.cards {
		if view == nil {
			continue
		}
		was := view.hovered
		view.hovered = (view == best)
		if view.hovered != was {
			changed = true
		}
	}
	if changed && h.ui.hand != nil {
		h.ui.hand.Layout()
	}
	return best
}

func (h *hand3DScene) Layout(cards []*cardView) {
	if h == nil {
		return
	}

	count := 0
	for _, view := range cards {
		if view != nil {
			count++
		}
	}
	if count == 0 {
		return
	}

	width := h.cardSpacing * float64(count-1)
	startX := -width / 2
	center := float64(count-1) / 2

	index := 0
	for _, view := range cards {
		if view == nil {
			continue
		}
		mesh := h.cards[view]
		if mesh == MeshInstance3D.Nil {
			index++
			continue
		}
		x := startX + float64(index)*h.cardSpacing
		y := 0.0
		if view.hovered && h.ui.dragging != view {
			y += h.cardLift
		}
		z := -h.baseDepth - math.Abs(float64(index)-center)*0.08
		mesh.AsNode3D().SetPosition(Vector3.XYZ{Float.X(x), Float.X(y), Float.X(z)})
		mesh.AsNode3D().SetRotationDegrees(Euler.Degrees{
			X: Angle.Degrees(0),
			Y: Angle.Degrees(Float.X((float64(index)-center)*h.cardTilt + h.cameraYaw)),
			Z: Angle.Degrees(0),
		})
		index++
	}
}
