package k1

import (
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

const (
	userkey = "user"
)

// Thanks to otraore for the code example
// https://gist.github.com/otraore/4b3120aa70e1c1aa33ba78e886bb54f3

func engine() *gin.Engine {
	r := gin.New()
	r.Use(sessions.Sessions("mysession", sessions.NewCookieStore([]byte("secret"))))
	r.Any("/login", login)
	r.GET("/logout", logout)

	return r
}

// AuthRequired is a simple middleware to check the session
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		// Abort the request with the appropriate error code
		//c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		c.Redirect(http.StatusFound, "/login"+"?next="+url.QueryEscape(c.Request.RequestURI))
		return
	}
	// Continue down the chain to handler etc
	c.Next()
}

// login is a handler that parses a form and checks for specific data
func login(c *gin.Context) {
	if c.Request.Method == "GET" {
		tpl := `
<body>
<div class="main">
    <div class="title">
        <span>密码登录</span>
    </div>
 
    <div class="title-msg">
        <span>请输入登录账户和密码</span>
    </div>
 
    <form class="login-form" method="post" novalidate >
        <!--输入框-->
        <div class="input-content">
            <!--autoFocus-->
            <div>
                <input type="text" autocomplete="off"
                       placeholder="用户名" name="username" required/>
            </div>
 
            <div style="margin-top: 16px">
                <input type="password"
                       autocomplete="off" placeholder="登录密码" name="password" required maxlength="32"/>
            </div>
        </div>
 
        <!--登入按钮-->
        <div style="text-align: center">
            <button type="submit" class="enter-btn" >登录</button>
        </div>
    </form>
 <style>
    body{
        background: #353f42;
    }
 
    *{
        padding: 0;
        margin: 0;
    }
 
    .main {
        margin: 0 auto;
        padding-left: 25px;
        padding-right: 25px;
        padding-top: 15px;
        width: 350px;
        height: 350px;
        background: #FFFFFF;
        /*以下css用于让登录表单垂直居中在界面,可删除*/
        position: absolute;
        top: 50%;
        left: 50%;
        margin-top: -175px;
        margin-left: -175px;
    }
 
    .title {
        width: 100%;
        height: 40px;
        line-height: 40px;
    }
 
    .title span {
        font-size: 18px;
        color: #353f42;
    }
 
    .title-msg {
        width: 100%;
        height: 64px;
        line-height: 64px;
    }
 
    .title:hover{
        cursor: default	;
    }
 
    .title-msg:hover{
        cursor: default	;
    }
 
    .title-msg span {
        font-size: 12px;
        color: #707472;
    }
 
    .input-content {
        width: 100%;
        height: 120px;
    }
 
    .input-content input {
        width: 330px;
        height: 40px;
        border: 1px solid #dad9d6;
        background: #ffffff;
        padding-left: 10px;
        padding-right: 10px;
    }
 
    .enter-btn {
        width: 350px;
        height: 40px;
        color: #fff;
        background: #0bc5de;
        line-height: 40px;
        text-align: center;
        border: 0px;
    }
 
    .foor{
        width: 100%;
        height: auto;
        color: #9b9c98;
        font-size: 12px;
        margin-top: 20px;
    }
 
    .enter-btn:hover {
        cursor:pointer;
        background: #1db5c9;
    }
 
    .foor div:hover {
        cursor:pointer;
        color: #484847;
        font-weight: 600;
    }
 
    .left{
        float: left;
    }
    .right{
        float: right;
    }
 </style>
</div>

</body>
`
		c.Writer.WriteString(tpl)
		c.Status(200)
		return
	}

	session := sessions.Default(c)

	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
		return
	}

	// Check for username and password match, usually from a database
	if username != "admin" || password != "admin" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// Save the username in the session
	session.Set(userkey, username) // In real world usage you'd set this to the users ID
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	if next := c.Param("next"); next != "" {
		c.Redirect(http.StatusFound, next)
	} else {
		c.Redirect(http.StatusFound, "/")
	}
	//c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}
	session.Delete(userkey)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.Redirect(http.StatusFound,"/")
	//c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func me(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "You are logged in"})
}
