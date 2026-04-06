package rpc

//go:generate colgen -imports=workout/pkg/workout -funcpkg=workout
//colgen:Training,Approach,Exercise,Category
//colgen:Training:MapP(workout)
//colgen:Approach:MapP(workout)
//colgen:Exercise:MapP(workout)
//colgen:Category:MapP(workout)
