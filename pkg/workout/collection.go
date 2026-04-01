package workout

//go:generate colgen --imports=workoutbot/pkg/db
//colgen:SiteUser,Training,Approach
//colgen:SiteUser:MapP(db)
//colgen:Training:MapP(db),Index(Date)
//colgen:Approach:MapP(db)
