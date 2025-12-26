package godot

import (
	"math"
	"sort"

	"graphics.gd/classdb/BaseMaterial3D"
	"graphics.gd/classdb/Camera3D"
	"graphics.gd/classdb/DirectionalLight3D"
	"graphics.gd/classdb/GUI"
	"graphics.gd/classdb/Image"
	"graphics.gd/classdb/ImageTexture"
	"graphics.gd/classdb/Label3D"
	"graphics.gd/classdb/MeshInstance3D"
	"graphics.gd/classdb/Node"
	"graphics.gd/classdb/Node3D"
	"graphics.gd/classdb/PlaneMesh"
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

// hand3DScene builds a simple 3D representation of the hand using a subviewport.
type hand3DScene struct {
	ui       *CardGameUI
	viewport SubViewport.Instance
	root     Node3D.Instance
	camera   Camera3D.Instance
	light    DirectionalLight3D.Instance

	cardMat         StandardMaterial3D.Instance
	cardTex         Texture2D.Instance
	avatarMat       StandardMaterial3D.Instance
	avatarTex       Texture2D.Instance
	inHand          map[*cardView]struct{}
	cardWidth       Float.X
	cardHeight      Float.X
	cardDepth       Float.X
	handRaise       Float.X
	handRaiseTarget Float.X
	raiseStep       Float.X
	motionLerp      Float.X
}

func loadCardTexture() Texture2D.Instance {
	paths := []string{
		"graphics/textures/card.png",
		"textures/card.png",
	}
	for _, p := range paths {
		if img := Image.LoadFromFile(p); img != Image.Nil {
			tex := ImageTexture.CreateFromImage(img)
			return tex.AsTexture2D()
		}
	}
	return Texture2D.Nil
}

func loadAvatarTexture() Texture2D.Instance {
	paths := []string{
		"graphics/textures/avatar.png",
		"textures/avatar.png",
	}
	for _, p := range paths {
		if img := Image.LoadFromFile(p); img != Image.Nil {
			tex := ImageTexture.CreateFromImage(img)
			return tex.AsTexture2D()
		}
	}
	return Texture2D.Nil
}

func newHand3DScene(ui *CardGameUI, vp SubViewport.Instance) *hand3DScene {
	if ui == nil || vp == SubViewport.Nil {
		return nil
	}

	root := Node3D.New()
	vp.AsNode().AddChild(root.AsNode())

	camera := Camera3D.New()
	camera.AsNode3D().SetPosition(Vector3.XYZ{0, 1.25, 4.2})
	camera.AsNode3D().LookAt(Vector3.XYZ{0, 0.1, 0})
	camera.SetCurrent(true)
	root.AsNode().AddChild(camera.AsNode())

	light := DirectionalLight3D.New()
	light.AsNode3D().SetRotation(Euler.Radians{X: Angle.Radians(-0.35), Y: 0, Z: 0})
	root.AsNode().AddChild(light.AsNode())

	mat := StandardMaterial3D.New()
	base := mat.AsBaseMaterial3D()
	base.SetAlbedoColor(Color.RGBA{R: 0.9, G: 0.9, B: 0.95, A: 1})
	base.SetMetallic(0.02)
	base.SetRoughness(0.25)
	base.SetCullMode(BaseMaterial3D.CullDisabled)

	tex := loadCardTexture()
	if tex != Texture2D.Nil {
		base.SetAlbedoTexture(tex)
	}

	avatarMat := StandardMaterial3D.New()
	avatarBase := avatarMat.AsBaseMaterial3D()
	avatarBase.SetAlbedoColor(Color.RGBA{R: 0.4, G: 0.45, B: 0.55, A: 1})
	avatarBase.SetMetallic(0.05)
	avatarBase.SetRoughness(0.6)
	avatarBase.SetCullMode(BaseMaterial3D.CullDisabled)

	avatarTex := loadAvatarTexture()
	if avatarTex != Texture2D.Nil {
		avatarBase.SetAlbedoTexture(avatarTex)
	}

	return &hand3DScene{
		ui:         ui,
		viewport:   vp,
		root:       root,
		camera:     camera,
		light:      light,
		cardMat:    mat,
		cardTex:    tex,
		avatarMat:  avatarMat,
		avatarTex:  avatarTex,
		inHand:     make(map[*cardView]struct{}),
		cardWidth:  1.0,
		cardHeight: 1.45,
		cardDepth:  0.02,
		raiseStep:  0.7,
		motionLerp: 0.18,
	}
}

func (h *hand3DScene) ready() bool {
	if h == nil || h.viewport == SubViewport.Nil || h.root == Node3D.Nil || h.camera == Camera3D.Nil {
		return false
	}
	return h.viewport.AsNode().IsInsideTree()
}

func (h *hand3DScene) rootNode() Node3D.Instance {
	if h == nil {
		return Node3D.Nil
	}
	return h.root
}

func (h *hand3DScene) ensureMesh(view *cardView) MeshInstance3D.Instance {
	if h == nil || view == nil {
		return MeshInstance3D.Nil
	}
	if view.mesh != MeshInstance3D.Nil {
		return view.mesh
	}
	mesh := MeshInstance3D.New()
	plane := PlaneMesh.New()
	plane.SetSize(Vector2.XY{h.cardWidth, h.cardHeight})
	plane.SetOrientation(PlaneMesh.FaceZ)
	mesh.SetMesh(plane.AsMesh())
	mesh.SetSurfaceOverrideMaterial(0, h.cardMat.AsMaterial())
	mesh.AsNode3D().SetPosition(Vector3.XYZ{0, h.cardHeight / 2, 0})
	mesh.AsNode3D().SetRotation(Euler.Radians{})
	h.root.AsNode().AddChild(mesh.AsNode())
	view.mesh = mesh

	overlay := Node3D.New()
	overlay.AsNode3D().SetPosition(Vector3.XYZ{0, 0, 0.03})
	mesh.AsNode().AddChild(overlay.AsNode())

	avatar := MeshInstance3D.New()
	avatarPlane := PlaneMesh.New()
	avatarPlane.SetSize(Vector2.XY{0.55, 0.55})
	avatarPlane.SetOrientation(PlaneMesh.FaceZ)
	avatar.SetMesh(avatarPlane.AsMesh())
	if h.avatarMat != StandardMaterial3D.Nil {
		avatar.SetSurfaceOverrideMaterial(0, h.avatarMat.AsMaterial())
	}
	avatar.AsNode3D().SetPosition(Vector3.XYZ{0, -0.05, 0})
	overlay.AsNode().AddChild(avatar.AsNode())

	label := Label3D.New()
	cardName := "Card"
	if view.instance != nil {
		cardName = view.instance.GetName()
	}
	label.SetText(cardName)
	label.SetFontSize(22)
	label.SetPixelSize(0.006)
	label.SetHorizontalAlignment(GUI.HorizontalAlignmentCenter)
	label.SetVerticalAlignment(GUI.VerticalAlignmentTop)
	label.SetWidth(h.cardWidth * 0.9)
	label.SetModulate(Color.RGBA{R: 0.95, G: 0.95, B: 0.98, A: 1})
	label.AsNode3D().SetPosition(Vector3.XYZ{0, h.cardHeight * 0.45, 0})
	overlay.AsNode().AddChild(label.AsNode())

	return mesh
}

func (h *hand3DScene) Add(view *cardView) {
	if h == nil || view == nil {
		return
	}
	h.ensureMesh(view)
	h.inHand[view] = struct{}{}
	view.location = "hand"
	view.fieldIndex = -1
	h.Layout()
}

func (h *hand3DScene) Detach(view *cardView) {
	if h == nil || view == nil {
		return
	}
	delete(h.inHand, view)
	h.Layout()
}

func (h *hand3DScene) Remove(view *cardView) {
	if h == nil || view == nil {
		return
	}
	delete(h.inHand, view)
	if view.mesh != MeshInstance3D.Nil {
		if parent := view.mesh.AsNode().GetParent(); parent != Node.Nil {
			parent.RemoveChild(view.mesh.AsNode())
		}
		view.mesh = MeshInstance3D.Nil
	}
	h.Layout()
}

func (h *hand3DScene) cardList() []*cardView {
	out := make([]*cardView, 0, len(h.inHand))
	for v := range h.inHand {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].instance.GetId() < out[j].instance.GetId()
	})
	return out
}

func (h *hand3DScene) Layout() {
	if h == nil || !h.ready() {
		return
	}
	cards := h.cardList()
	if len(cards) == 0 {
		return
	}
	count := float64(len(cards))
	maxSpread := 3.6
	spacing := 0.65
	if len(cards) > 1 {
		spacing = maxSpread / (count - 1)
		if spacing > 0.85 {
			spacing = 0.85
		}
		if spacing < 0.55 {
			spacing = 0.55
		}
	}
	startX := -float64(len(cards)-1) * spacing / 2
	baseZ := Float.X(2.15)
	arcZ := 0.18
	arcY := 0.08
	baseY := Float.X(-1.0) + h.handRaise*h.raiseStep
	for i, view := range cards {
		if view.mesh == MeshInstance3D.Nil {
			continue
		}
		var norm float64
		center := (count - 1) / 2
		if center > 0 {
			norm = (float64(i) - center) / center
		}
		x := Float.X(startX + float64(i)*spacing)
		y := baseY + Float.X(arcY*(1-math.Abs(norm)))
		z := baseZ - Float.X(arcZ*(1-math.Abs(norm)))
		if view.hovered {
			y += 0.2
			z += 0.1
		}
		targetPos := Vector3.XYZ{x, y, z}
		currentPos := view.mesh.AsNode3D().Position()
		view.mesh.AsNode3D().SetPosition(lerpVec3(currentPos, targetPos, h.motionLerp))

		targetRot := Euler.Radians{X: Angle.Radians(-0.08), Y: 0, Z: 0}
		currentRot := view.mesh.AsNode3D().Rotation()
		view.mesh.AsNode3D().SetRotation(lerpEuler(currentRot, targetRot, h.motionLerp))
	}
}

func (h *hand3DScene) SetHandRaise(value Float.X) {
	if h == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	h.handRaiseTarget = value
}

// HoverAt raycasts against the hand cards and returns the hovered view.
func (h *hand3DScene) HoverAt(pointer Vector2.XY) *cardView {
	if h == nil || !h.ready() {
		return nil
	}
	origin := h.camera.ProjectRayOrigin(pointer)
	dir := h.camera.ProjectRayNormal(pointer)

	var hovered *cardView
	best := math.MaxFloat64
	for view := range h.inHand {
		if view.mesh == MeshInstance3D.Nil {
			continue
		}
		center := view.mesh.AsNode3D().Position()
		rot := view.mesh.AsNode3D().Rotation()
		yaw := float64(rot.Y)

		cy, sy := math.Cos(yaw), math.Sin(yaw)
		nx := sy
		ny := 0.0
		nz := cy

		den := float64(dir.X)*nx + float64(dir.Y)*ny + float64(dir.Z)*nz
		if math.Abs(den) < 1e-4 {
			continue
		}
		t := (float64(center.X-origin.X)*nx + float64(center.Y-origin.Y)*ny + float64(center.Z-origin.Z)*nz) / den
		if t < 0 {
			continue
		}
		hit := Vector3.XYZ{
			origin.X + dir.X*Float.X(t),
			origin.Y + dir.Y*Float.X(t),
			origin.Z + dir.Z*Float.X(t),
		}

		hx := float64(hit.X - center.X)
		hy := float64(hit.Y - center.Y)
		hz := float64(hit.Z - center.Z)

		localX := hx*cy + hz*sy
		localZ := -hx*sy + hz*cy

		if math.Abs(localX) > float64(h.cardWidth)/2+0.05 {
			continue
		}
		if math.Abs(hy) > float64(h.cardHeight)/2+0.05 {
			continue
		}
		if math.Abs(localZ) > float64(h.cardDepth)/2+0.05 {
			continue
		}

		if t < best {
			best = t
			hovered = view
		}
	}

	for view := range h.inHand {
		view.hovered = view == hovered
	}
	h.Layout()
	return hovered
}

func (h *hand3DScene) Update() {
	if h == nil || !h.ready() {
		return
	}
	if h.handRaise != h.handRaiseTarget {
		h.handRaise = Float.X(lerpFloat(float64(h.handRaise), float64(h.handRaiseTarget), float64(h.motionLerp)))
	}
	h.Layout()
}

func lerpFloat(a, b, t float64) float64 {
	return a + (b-a)*t
}

func lerpVec3(current, target Vector3.XYZ, t Float.X) Vector3.XYZ {
	return Vector3.XYZ{
		X: Float.X(lerpFloat(float64(current.X), float64(target.X), float64(t))),
		Y: Float.X(lerpFloat(float64(current.Y), float64(target.Y), float64(t))),
		Z: Float.X(lerpFloat(float64(current.Z), float64(target.Z), float64(t))),
	}
}

func lerpEuler(current, target Euler.Radians, t Float.X) Euler.Radians {
	return Euler.Radians{
		X: Angle.Radians(lerpFloat(float64(current.X), float64(target.X), float64(t))),
		Y: Angle.Radians(lerpFloat(float64(current.Y), float64(target.Y), float64(t))),
		Z: Angle.Radians(lerpFloat(float64(current.Z), float64(target.Z), float64(t))),
	}
}
