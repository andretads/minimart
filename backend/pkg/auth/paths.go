package auth

// PublicRoutePaths lists the routes that need neither an access token nor a
// permission. A path ending in "/" is matched as a prefix; every other path
// is matched exactly.
//
// The list is shared by the server wiring and the route-policy coverage test,
// so a public route cannot be public in one layer and protected in the other.
// The "/swagger/" tree is always public and is therefore not listed here.
var PublicRoutePaths = []string{
	"/livez",
	"/readyz",
	"/metrics",
	"/v1/auth/signup",
	"/v1/auth/login",
	"/v1/auth/refresh",
	"/v1/auth/logout",
	"/v1/showcase/products",
	"/v1/showcase/products/",
	"/v1/showcase/categories",
}
