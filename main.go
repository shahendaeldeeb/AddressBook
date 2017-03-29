package main

import (
	"net/http"
	"html/template"
	"github.com/gorilla/mux"
	"os"
	"log"
	"strings"
	"bufio"
	"database/sql"
	//"github.com/coopernurse/gorp"
	//"encoding/json"
	_"github.com/go-sql-driver/mysql"
  	"github.com/goincremental/negroni-sessions"
	//"github.com/goincremental/negroni-sessions/cookiestore"
	"github.com/urfave/negroni"
	"golang.org/x/crypto/bcrypt"
	"github.com/goincremental/negroni-sessions/cookiestore"
	"fmt"
//	"os/user"
	"strconv"
	//"golang.org/x/tools/go/gcimporter15/testdata"
	"encoding/json"
)
type User struct {
	 Username string `db:"username"`
	 Password []byte `db:"userpassword"`
}
type contact_info struct{
	Name   		string   `db:"name"`
	Number 		string   `db:"number"`
	Email  		string   `db:"email"`
	Nationality 	string   `db:"nationality"`
	Address 	string   `db:"address"`
	Username  	string   `db:"username"`
	ID 		int   `db:"id"`
}
type LoginPage struct {
	Error string
}
type page struct{
	Contacts []contact_info
	Numbers []Telephone
}
type Telephone struct {
	Contact_id int `db:"contactid"`
	Number string  	`db:"number"`
	Num_id int	`db:"numid"`

}
var muxer = mux.NewRouter()
func main(){
	initDb()

	var contact contact_info
	var userAccount User
	var telephone Telephone

	muxer.HandleFunc("/viewNumbers/{id}", telephone.ViewTelephonesHandler).Methods("GET")
	muxer.HandleFunc("/contact",contact.SaveContactHandler).Methods("POST")
	muxer.HandleFunc("/contact/{id}" , contact.DeleteContactHandler).Methods("DELETE")
	muxer.HandleFunc("/deleteNumber/{numId}" , telephone.DeleteNumberHandler).Methods("DELETE")
	muxer.HandleFunc("/addNumber/{ContactId}" ,telephone.AddNumberHandler).Methods("POST")
 	muxer.HandleFunc("/logout" ,userAccount.logoutHandler)
	muxer.HandleFunc("/" ,serverContent)

	muxer.HandleFunc("/login" ,userAccount.loginContactHandler)
	muxer.HandleFunc("/{page_alias}" , serverContent)

	muxer.HandleFunc("/img/" ,serverResource)
	muxer.HandleFunc("/js/" ,serverResource)
	muxer.HandleFunc("/css/{page_alias}" ,serverResource)

	//it provides some default middleware
	n := negroni.Classic()
	n.Use(sessions.Sessions("go-for-web-dev" , cookiestore.New([]byte("my-secret-123"))))
	//add Handler to middleware stack
	n.Use(negroni.HandlerFunc(verifyDatabase))
	// to add http.Handler (process that runs in response to request made to web app.)alli f el mux in negroni stack
	n.UseHandler(muxer)
	n.Run(":8080")
	defer db.Close()
}
var db *sql.DB
//var dbmap *gorp.DbMap
func initDb (){
	db,_ =sql.Open("mysql" , "root:shahenda_hassan@/mydatabase")
      	//first parameter is a pointer to our database , second is used sql (sqlight or mysql)
	//dbmap = &gorp.DbMap{Db:db , Dialect:gorp.MySQLDialect{"InnoDB", "UTF8"}}

	//dbmap.AddTableWithName(contact_info{} , "Contacts").SetKeys(false , "name")
	//dbmap.AddTableWithName(Telephone{} , "Telephones").SetKeys(false , "number")
	//dbmap.AddTableWithName(User{} , "Users").SetKeys(false , "username")

	//dbmap.CreateTablesIfNotExists()
}

func verifyDatabase(w http.ResponseWriter , r *http.Request , next http.HandlerFunc){
	//ping() -> it verify connection to database is still alive , establishing connection if necessary
	err := db.Ping();
	if err != nil{
		http.Error(w, err.Error() , http.StatusInternalServerError)
		return
	}
	next(w,r)
}

func(info User)loginContactHandler (w http.ResponseWriter , r *http.Request){
	var p LoginPage

	if r.FormValue("signUp") != ""{
		secret , _ := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")) , bcrypt.DefaultCost)
		info = User{r.FormValue("username") , secret}
		stmt ,err := db.Prepare("INSERT Users SET username=? , userpassword=?")
		_ ,err =stmt.Exec(info.Username ,info.Password)
		if err != nil{
			p.Error = err.Error()

		}else{
			//sessions.getsession() to create session variable
			//.set key-> User , value ->user.Username
			sessions.GetSession(r).Set("User" , info.Username)
			http.Redirect(w, r, "/Home" , http.StatusFound)
			return
		}

	}else if r.FormValue("login") != ""{
		row ,err := db.Query("SELECT * FROM Users WHERE username =?" , r.FormValue("username"))
		for  row.Next(){
			row.Scan(&info.Username , &info.Password)
		}
		defer row.Close()
		if err != nil {
			p.Error = err.Error()
			return
		}else if row == nil{
			p.Error = " No such user found with the username : " + r.FormValue("username")
		}else {
			err := bcrypt.CompareHashAndPassword(info.Password , []byte(r.FormValue("password")))
			if err != nil{
				p.Error = err.Error()
			}else {
				sessions.GetSession(r).Set("User" , info.Username)
				http.Redirect(w, r, "/" , http.StatusFound)
				return
			}
		}
	}



	templates := template.Must(template.ParseFiles("Login.html"))
	err:= templates.Execute(w,p)
	if err != nil {
		http.Error(w, err.Error() , http.StatusInternalServerError)
		return
	}



 	 //info.username = r.FormValue("username")
	 //info.password = r.FormValue("password")
	//http.Redirect(w, r, "/Home" , http.StatusFound)
}
func(u User)logoutHandler(w http.ResponseWriter , r *http.Request){
	sessions.GetSession(r).Set("User" , nil)
	http.Redirect(w , r , "/Login" , http.StatusFound)
}
func(contact contact_info)SaveContactHandler(w http.ResponseWriter , r *http.Request){
	contact.Name = r.FormValue("name")
	contact.Number = r.FormValue("number")
	contact.Email = r.FormValue("email")
	contact.Nationality = r.FormValue("nationality")
	contact.Address = r.FormValue("address")
	contact.Username = sessions.GetSession(r).Get("User").(string)

	stmt , err := db.Prepare("INSERT Contacts SET name=?  , email=? , nationality=? ,address=? ,username=?")
	checkErr(err)
	res , err := stmt.Exec(contact.Name,contact.Email,contact.Nationality,contact.Address,contact.Username)
	checkErr(err)
	id , err := res.LastInsertId()
	checkErr(err)

	var num Telephone
	num.Number = r.FormValue("number")

	num.Contact_id = int(id)

	_ =addNum(num.Number , num.Contact_id)
	sessions.GetSession(r).Set("Contact" , id)
}
func(contact contact_info)DeleteContactHandler (w http.ResponseWriter , r *http.Request){
	ID , _ := strconv.ParseInt(mux.Vars(r)["id"] , 10 , 64)
	fmt.Println(ID)
	stmt , err :=db.Prepare("delete from Contacts where id=?")
	checkErr(err)
	_ ,err = stmt.Exec(ID)
	checkErr(err)

}
func(tele Telephone)ViewTelephonesHandler(w http.ResponseWriter , r *http.Request){
	ID , _ := strconv.ParseInt(mux.Vars(r)["id"] , 10 , 64)
	p1:=page{Numbers:[]Telephone{} }
	//fmt.Println(ID)
	rows,err := db.Query("select * from Telephones where contactid =?" , ID)
	checkErr(err)
	for rows.Next() {
		rows.Scan(&tele.Contact_id ,&tele.Number , &tele.Num_id)
		p1.Numbers = append(p1.Numbers , tele)
	}
	//fmt.Println(p1.Numbers)
	encoder := json.NewEncoder(w)
	err = encoder.Encode(p1.Numbers)

	if err != nil{
		http.Error(w, err.Error() , http.StatusInternalServerError)
	}
}
func(tele Telephone)DeleteNumberHandler(w http.ResponseWriter , r *http.Request){
	NumID , _ := strconv.ParseInt(mux.Vars(r)["numId"] , 10 , 64)
	fmt.Println(NumID)
	stmt , err :=db.Prepare("delete from Telephones where Numid=?")
	checkErr(err)
	_ ,err = stmt.Exec(NumID)
	checkErr(err)

}
func(tele Telephone)AddNumberHandler(w http.ResponseWriter , r *http.Request){
	fmt.Println("helloo from add number ")
	ContactID , _ := strconv.ParseInt(mux.Vars(r)["ContactId"] , 10 , 64)
	//fmt.Print("contactid")
	fmt.Println(ContactID)
	fmt.Print("new number :")

	fmt.Println(r.FormValue("NewNumber"))
	id := addNum(r.FormValue("NewNumber") , int(ContactID))
	tele.Number = r.FormValue("NewNumber")
	tele.Contact_id = int(ContactID)
	tele.Num_id = int(id)
	encoder := json.NewEncoder(w)
	err := encoder.Encode(tele)

	if err != nil{
		http.Error(w, err.Error() , http.StatusInternalServerError)
	}

}
func addNum (number string , contactId int )(int64){
	stmt2,err :=db.Prepare("INSERT Telephones SET number=? , contactid=?")
	checkErr(err)
	 stmt, err := stmt2.Exec(number , contactId)
	checkErr(err)
	id ,err :=stmt.LastInsertId()
	checkErr(err)
	return id
}


//-------------------------------Static Pages Handle Functions ---------------------------------------------
//retrieve all static pages
func populateStaticPages() *template.Template{
	result := template.New("templates")
	templatePathes := new([]string)
	basepath:= "pages"
	templateFolder , _ := os.Open(basepath)
	defer templateFolder.Close()
	//READdir-> to read directory name and return a list of directory entries
	// it takes -1 to return all files in single slice
	templatePathRaw , _ := templateFolder.Readdir(-1)
	for _ , pathinfo := range templatePathRaw {
		log.Println(pathinfo.Name())
		*templatePathes = append(*templatePathes ,basepath+"/"+pathinfo.Name())
	}
	//parsefile-> it parses source code and return corresponding file node
	result.ParseFiles(*templatePathes...)
	return  result
}
func serverContent (w http.ResponseWriter , r *http.Request){
	staticPages := populateStaticPages()

	p := page{Contacts:[]contact_info{}}
	rows , err := db.Query("select * from Contacts where username =? " ,sessions.GetSession(r).Get("User").(string))
        checkErr(err)
	var contact contact_info
	for rows.Next() {
		rows.Scan(&contact.Name ,&contact.Email ,&contact.Nationality ,&contact.Address , &contact.Username ,&contact.ID)
		p.Contacts = append(p.Contacts ,contact)
	}
	//mux.vars() -> creates a map of rout variables that can be retrieved
        urlParams := mux.Vars(r)
	page_alias := urlParams["page_alias"]
	if page_alias ==""{
		page_alias="Home"
	}
	staticPage := staticPages.Lookup(page_alias+".html")
	if page_alias == "Home"{
		fmt.Println("before execute")
		staticPage.Execute(w,p)
	}else{
	staticPage.Execute(w , nil)}


}
// it serve file of type img , js , css
func serverResource(w http.ResponseWriter , r *http.Request){
	path := "public/bs4"  + r.URL.Path
	var contentType string
	if strings.HasSuffix(path , ".css"){
		contentType="text/css; char-set=utf-8"
	}else if strings.HasSuffix(path , ".png"){
		contentType="image/png; char-set=utf-8"
	}else if strings.HasSuffix(path , ".jpg"){
		contentType="image/jpg; char-set=utf-8"
	}else if strings.HasSuffix(path , ".js"){
		contentType="application/javascript; char-set=utf-8"
	}else {
		contentType="text/plain; char-set=utf-8"
	}
	//log.Println(path)
	f,err := os.Open(path)
	if err == nil {
		defer f.Close()
		w.Header().Add("content-type" , contentType)
		br := bufio.NewReader(f)
		br.WriteTo(w)
	}else {
		w.WriteHeader(404)
	}
}
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}