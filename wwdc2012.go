package main

func init() {
	RegisterEvent(&Event{
		Id: "wwdc2012",
		Name: "WWDC 2012",
		Resolver: func(event *Event) []*Session {
			return getSessionsFromUrl(event, "https://developer.apple.com/videos/wwdc/2012/")
		},
	})
}
