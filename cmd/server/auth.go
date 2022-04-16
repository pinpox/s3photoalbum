package main
import (

	"golang.org/x/crypto/bcrypt"
    "errors"
    "fmt"
    "github.com/golang-jwt/jwt"

	"github.com/gin-gonic/gin"
	"net/http"
    "time"
)




func verifyToken(c *gin.Context) {

	token, err := c.Cookie("token")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{})
		return
	}

	id, username, err := validateToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{})
		return
	}

	c.Set("id", id)
	c.Set("username", username)
	c.Next()
}


func hashAndSalt(pwd string) (string, error) {

    // Store this "hash" somewhere, e.g. in your database
    // Use GenerateFromPassword to hash & salt pwd.
    // MinCost is just an integer constant provided by the bcrypt
    // package along with DefaultCost & MaxCost. 
    // The cost can be any value you want provided it isn't lower
    // than the MinCost (4)
    hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
    if err != nil {
			return "", err

    }
    // GenerateFromPassword returns a byte slice so we need to
    // convert the bytes to a string and return it
    return string(hash), nil
}


type authClaims struct {
    jwt.StandardClaims
    UserID uint `json:"userId"`
}
func generateToken(user User) (string, error) {
    expiresAt := time.Now().Add(24 * time.Hour).Unix()
    token := jwt.NewWithClaims(jwt.SigningMethodHS512, authClaims{
        StandardClaims: jwt.StandardClaims{
            Subject:   user.Username,
            ExpiresAt: expiresAt,
        },
        UserID: user.ID,
    })
    tokenString, err := token.SignedString(jwtKey)
    if err != nil {
        return "", err
    }
    return tokenString, nil
}

func validateToken(tokenString string) (uint, string, error) {
    var claims authClaims
    token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtKey, nil
    })
    if err != nil {
        return 0, "", err
    }
    if !token.Valid {
        return 0, "", errors.New("invalid token")
    }
    id := claims.UserID
    username := claims.Subject
    return id, username, nil
}
