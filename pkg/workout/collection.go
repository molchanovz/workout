package workout

//go:generate colgen --imports=workoutbot/pkg/db
//colgen:BotUser,Training
//colgen:BotUser:MapP(db)
//colgen:Training:MapP(db),Index(Date)
