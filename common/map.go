package common

import v1 "overseer/build/go"

type MapTranslation struct {
	X         int64
	XPositive bool
	Y         int64
	YPositive bool
}

func (t MapTranslation) IsCoordinate(origin *v1.MapPosition, target *v1.MapPosition) bool {
	x := origin.X
	y := origin.Y

	if t.XPositive {
		x += t.X
	} else {
		x -= t.X
	}

	if t.YPositive {
		y += t.Y
	} else {
		y -= t.Y
	}

	return x == target.X && y == target.Y
}

func GetDirection(origin *v1.MapPosition, target *v1.MapPosition) string {
	if origin.X == target.X && origin.Y == target.Y {
		return "none"
	}

	if origin.X == target.X {
		if origin.Y < target.Y {
			return "north"
		} else {
			return "south"
		}
	}

	if origin.Y == target.Y {
		if origin.X < target.X {
			return "east"
		} else {
			return "west"
		}
	}

	if origin.X < target.X {
		if origin.Y < target.Y {
			return "north-east"
		} else {
			return "south-east"
		}
	}

	if origin.Y < target.Y {
		return "north-west"
	}

	return "south-west"
}

func FindNeighbors(origin *v1.MapPosition, grid map[string]*v1.MapCoordinateDetail) []*v1.MapCoordinateDetail {
	neighbors := make([]*v1.MapCoordinateDetail, 0)

	for _, translation := range AllPossibleDirectTranslations {
		for _, coordinate := range grid {
			if coordinate != nil {
				if translation.IsCoordinate(origin, coordinate.Position) {
					neighbors = append(neighbors, coordinate)
				}
			}
		}
	}

	return neighbors
}

var (
	// these are translation coordinates that allow for easier lookup of neighbors
	SouthernNeighborTranslation     = MapTranslation{X: 0, XPositive: false, Y: 1, YPositive: false}
	EasternNeighborTranslation      = MapTranslation{X: 1, XPositive: true, Y: 0, YPositive: false}
	NorthernNeighborTranslation     = MapTranslation{X: 0, XPositive: false, Y: 1, YPositive: true}
	WesternNeighborTranslation      = MapTranslation{X: 1, XPositive: false, Y: 0, YPositive: true}
	SouthWesternNeighborTranslation = MapTranslation{X: 1, XPositive: false, Y: 1, YPositive: false}
	SouthEasternNeighborTranslation = MapTranslation{X: 1, XPositive: true, Y: 1, YPositive: false}
	NorthWesternNeighborTranslation = MapTranslation{X: 1, XPositive: false, Y: 1, YPositive: true}
	NorthEasternNeighborTranslation = MapTranslation{X: 1, XPositive: true, Y: 1, YPositive: true}

	AllPossibleDirectTranslations = []MapTranslation{
		SouthernNeighborTranslation,
		EasternNeighborTranslation,
		NorthernNeighborTranslation,
		WesternNeighborTranslation,
		SouthWesternNeighborTranslation,
		SouthEasternNeighborTranslation,
		NorthWesternNeighborTranslation,
		NorthEasternNeighborTranslation,
	}
)
