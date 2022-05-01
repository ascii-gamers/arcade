package arcade

type DistanceVectorRouter struct {
	table *RoutingTable
}

func NewDistanceVectorRouter() *DistanceVectorRouter {
	return &DistanceVectorRouter{
		table: NewRoutingTable(),
	}
}
