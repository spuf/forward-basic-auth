package design

import (
	. "goa.design/goa/v3/dsl"
)

var _ = API("forward-basic-auth", func() {
	Title("forward-basic-auth")
	Version("1.0.0")
	Server("server", func() {
		Host("localhost", func() {
			URI("http://localhost:4013")
		})
	})
})

var UsersData = Type("UsersData", MapOf(String, String, func() {
	Key(func() {
		Pattern("^[^:]+$")
		Example("alice")
	})
	Elem(func() {
		MinLength(1)
		Example("p4ssw0rd")
	})
}))

var _ = Service("api", func() {
	Description("UsersData management.")
	Method("users", func() {
		Description("Update users.")
		Payload(UsersData)
		Result(Empty)

		HTTP(func() {
			PUT("/users")
			Body(UsersData)
			Response(StatusOK)
		})
	})
})

var BasicAuth = BasicAuthSecurity("basic_auth")
var AuthenticationError = Type("AuthenticationError", func() {
	Description("Authentication error.")
	Attribute("code", String, func() {
		Description("Authenticated error code.")
		Meta("struct:error:name")
		Enum(
			"password_is_incorrect",
			"username_does_not_exist",
		)
	})
	Attribute("message", String, func() {
		Description("Authenticated error message.")
		Example("The password is incorrect.")
	})
	Required("code", "message")
})

var _ = Service("authentication", func() {
	Method("auth", func() {
		Description("Basic authentication.")
		Security(BasicAuth)
		Payload(func() {
			Username("username", String)
			Password("password", String)
		})

		Error("username_does_not_exist", AuthenticationError, "The username does not exist.")
		Error("password_is_incorrect", AuthenticationError, "The password is incorrect.")
		Result(func() {
			Attribute("username", String, func() {
				Description("Authenticated username.")
				Example("alice")
			})
			Required("username")
		})

		HTTP(func() {
			GET("/auth")
			Response(StatusOK, func() {
				Header("username:X-Auth-User")
				Body(Empty)
			})
			Response("username_does_not_exist", StatusUnauthorized)
			Response("password_is_incorrect", StatusUnauthorized)
		})
	})
})
