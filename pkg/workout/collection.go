package workout

//go:generate colgen --imports=workout/pkg/db
//colgen:SiteUser,Training,Approach,Exercise,Category
//colgen:SiteUser:MapP(db)
//colgen:Training:MapP(db),Index(Date)
//colgen:Approach:MapP(db)
//colgen:Exercise:MapP(db)
//colgen:Category:MapP(db)
