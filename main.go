package main

import (
	"gopkg.in/validator.v2"
	"net/http"
	"html/template"
	"github.com/gorilla/mux"
	"os"
	"log"
	"strings"
	"bufio"
	"database/sql"
_	"github.com/go-sql-driver/mysql"
  	"github.com/goincremental/negroni-sessions"
	"github.com/urfave/negroni"
	"golang.org/x/crypto/bcrypt"
	"github.com/goincremental/negroni-sessions/cookiestore"
	//"fmt"
	"strconv"
	"encoding/json"
)
type User struct {
	 Username string ` validate:"regexp=^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$"   db:"username"`
	 Password []byte `validate: "nonzero" db:"userpassword"`
}
type ContactInfo struct{
	Id 		int      `db:"id"`
	Name   		string   ` validate:"nonzero" db:"name"`
	Number 		string   ` validate:"min=8 , max=12 , nonzero" db:"number"`
	Email  		string   ` validate:"nonzero,regexp=^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$ "db:"email"`
	Nationality 	string   ` validate :"nonzero" db:"nationality"`
	Address 	string   `validate:"nonzero" db:"address"`
	Username  	string   `db:"username"`

}
type LoginPage struct {
	Error string
}
type page struct{
	Contacts []ContactInfo
	Numbers []Telephone
}
type Telephone struct {
	ContactId int   `db:"contactid"`
	Number string  	`validate :"nonzero" db:"number"`
	Num_id int	`db:"numid"`

}
type HandlersVars struct {
	db *sql.DB
}
type middlewareHandlers struct{
	*HandlersVars
	NewMiddleWarHandler func(http.ResponseWriter, *http.Request ,http.HandlerFunc, *HandlersVars)
}
func (middleware middlewareHandlers)ServeHTTP( w http.ResponseWriter , r *http.Request , next http.HandlerFunc){
	// Updated to pass app.handlervars as a parameter to our handler type.
	middleware.NewMiddleWarHandler(w,r,next,middleware.HandlersVars)

}


func main() {
	muxer := mux.NewRouter()
	dataBase := initDb()
	defer dataBase.Close()
	usedDataBase := &HandlersVars{db: dataBase}

	muxer.HandleFunc("/viewnumbers/{id}", func(w http.ResponseWriter, r *http.Request)  {
		var tele Telephone
		p := tele.ViewTelephonesHandler(tele.ContactId,tele.Number,tele.Num_id,mux.Vars(r)["id"], usedDataBase)
		encoder := json.NewEncoder(w)
		err := encoder.Encode(p.Numbers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	muxer.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request){
		var contact ContactInfo
		contact.Name = r.FormValue("name")
		contact.Number = r.FormValue("number")
		contact.Email = r.FormValue("email")
		contact.Nationality = r.FormValue("nationality")
		contact.Address = r.FormValue("address")
		contact.Username = sessions.GetSession(r).Get("User").(string)
		if errs := validator.Validate(contact); errs != nil {
			checkErr(errs)
		}
		// TODO: Validate el data
		// TODO: Mini FLTR
		//       F Format
		//       L Length
		//       T Type
		//       R Range
		var num Telephone

		//id := contact.SaveContactHandler(&contact.Name , &contact.Email,&contact.Nationality,&contact.Address,&contact.Username,usedDataBase)
		id := contact.SaveContactHandler(&contact,usedDataBase)


		num.Number = r.FormValue("number")

		num.ContactId = int(id)

		_ = addNum(num.Number, num.ContactId, usedDataBase)
		sessions.GetSession(r).Set("Contact", id)
	})
	muxer.HandleFunc("/contact/{id}", func(w http.ResponseWriter, r *http.Request) {
		var contact ContactInfo
		contact.DeleteContactHandler(mux.Vars(r)["id"], usedDataBase)
	}).Methods("DELETE")
	muxer.HandleFunc("/deletenumber/{numId}", func(w http.ResponseWriter, r *http.Request) {
		var telephone Telephone
		telephone.DeleteNumberHandler(mux.Vars(r)["numId"], usedDataBase)
	}).Methods("DELETE")
	muxer.HandleFunc("/addnumber/{ContactId}", func(w http.ResponseWriter, r *http.Request) {
		var telephone Telephone
		telephone.Number = r.FormValue("NewNumber")
		if errs := validator.Validate(telephone); errs != nil {
			checkErr(errs)
		}
		ContactID,id := telephone.AddNumberHandler(r.FormValue("NewNumber"), mux.Vars(r)["ContactId"], usedDataBase)

		telephone.Num_id = int(id)
		telephone.ContactId = int(ContactID)
		encoder := json.NewEncoder(w)
		err := encoder.Encode(telephone)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("POST")
	muxer.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		sessions.GetSession(r).Set("User", nil)
		http.Redirect(w, r, "/Login", http.StatusFound)
	})
	muxer.HandleFunc("/", func(w http.ResponseWriter ,r*http.Request){
		p,page_alias, staticPage:=serverContent(mux.Vars(r),sessions.GetSession(r).Get("User").(string),usedDataBase)
		if page_alias == "Home"{
			staticPage.Execute(w,p)
		}else{

			staticPage.Execute(w , nil)

		}
	})
	muxer.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		info := User{Username:r.FormValue("username") ,Password:[]byte(r.FormValue("password")) }

		var userAccount User
		if errs := validator.Validate(info); errs != nil {
			checkErr(errs)
		}
		p,err:=userAccount.loginContactHandler(&info ,r.FormValue("login"), r.FormValue("signUp"), usedDataBase)

		if r.FormValue("signUp") != ""{
			if err != nil{
				p.Error = err.Error()

			}else{
				//sessions.getsession() to create session variable
				//.set key-> User , value ->user.Username
				sessions.GetSession(r).Set("User" , r.FormValue("username"))
				http.Redirect(w, r, "/Home" , http.StatusFound)
				return
			}
		} else if r.FormValue("login") != ""{
			if err != nil {
				p.Error = err.Error()
				return
			}else {
				err := bcrypt.CompareHashAndPassword(info.Password, []byte(r.FormValue("password")))
				if err != nil{
					p.Error = err.Error()
				}else {
					sessions.GetSession(r).Set("User" , r.FormValue("username"))
					http.Redirect(w, r, "/Home" , http.StatusFound)
					return
				}
			}
		}



		templates := template.Must(template.ParseFiles("Login.html"))

		err = templates.Execute(w,p)

		if err != nil {
			http.Error(w, err.Error() , http.StatusInternalServerError)
			return
		}

	}).Methods("GET")
	muxer.HandleFunc("/{page_alias}" , func(w http.ResponseWriter ,r*http.Request){
	p,page_alias, staticPage:=serverContent(mux.Vars(r),sessions.GetSession(r).Get("User").(string),usedDataBase)
		if page_alias == "Home"{
			staticPage.Execute(w,p)
		}else{

			staticPage.Execute(w , nil)

		}
	})
	muxer.HandleFunc("/img/" ,serverResource)
	muxer.HandleFunc("/js/" ,serverResource)
	muxer.HandleFunc("/css/{page_alias}" ,serverResource)


	//it provides some default middleware
	n := negroni.Classic()
	n.Use(sessions.Sessions("go-for-web-dev" , cookiestore.New([]byte("my-secret-123"))))
	//add Handler to middleware stack
	n.Use(middlewareHandlers{usedDataBase,verifyDatabase})
	// to add http.Handler (process that runs in response to request made to web app.)alli f el mux in negroni stack
	n.UseHandler(muxer)
	n.Run(":8080")

}
//var db *sql.DB
func initDb () *sql.DB{
	db, err := sql.Open("mysql" , "root:shahenda_hassan@/mydatabase")
	if err != nil {
		panic(err.Error())
	}
	return db

}
func verifyDatabase(w http.ResponseWriter , r *http.Request , next http.HandlerFunc , a *HandlersVars ){
	//ping() -> it verify connection to database is still alive , establishing connection if necessary
	err := a.db.Ping();
	if err != nil{
		http.Error(w, err.Error() , http.StatusInternalServerError)
		return
	}
	next(w,r)
}

func(info User)loginContactHandler (objuser *User  ,  login string , signUP string ,a *HandlersVars )(LoginPage,error) {
	var p LoginPage
	var err1 error
	var row *sql.Rows

	if signUP != ""{
		secret , _ := bcrypt.GenerateFromPassword(objuser.Password , bcrypt.DefaultCost)
		stmt ,err := a.db.Prepare("INSERT Users SET username=? , userpassword=?")
		_ , err =stmt.Exec(objuser.Username ,secret)
		err1=err

	}else if login  != ""{
		row ,err1 = a.db.Query("SELECT * FROM Users WHERE username =?" , objuser.Username )

		if row.Next(){
			row.Scan(&objuser.Username, &objuser.Password)
		}
		defer row.Close()

		 if row == nil{
			p.Error = " No such user found with the username : " + objuser.Username
		}


	}

	return p ,err1

}
func(contact ContactInfo)SaveContactHandler(contactInfo *ContactInfo , a *HandlersVars) int64 {

	stmt , err := a.db.Prepare("INSERT Contacts SET name=?  , email=? , nationality=? ,address=? ,username=?")
	checkErr(err)

	res , err := stmt.Exec(contactInfo.Name ,contactInfo.Email,contactInfo.Nationality,contactInfo.Address,contactInfo.Username)
	checkErr(err)
	id , err := res.LastInsertId()
	checkErr(err)

	return id

}
func(contact ContactInfo)DeleteContactHandler (id string ,a *HandlersVars){
	ID , _ := strconv.ParseInt(id , 10 , 64)
	stmt , err := a.db.Prepare("delete from Contacts where id=?")
	checkErr(err)
	_ ,err = stmt.Exec(ID)
	checkErr(err)

}
func(tele Telephone )ViewTelephonesHandler(teleContactId int , teleNumber string , teleNum_id int , id string, a *HandlersVars)page{
	ID , _ := strconv.ParseInt(id, 10 , 64)
	p1:=page{Numbers:[]Telephone{} }
	rows,err := a.db.Query("select * from Telephones where contactid =?" , ID)
	checkErr(err)
	//Telephone{ContactId: , Num_id:}
	for rows.Next() {
		rows.Scan(&teleContactId,&teleNumber, &teleNum_id)
		p1.Numbers = append(p1.Numbers , Telephone{ ContactId:teleContactId ,Number:teleNumber, Num_id:teleNum_id })
	}
	return p1

}
func(tele Telephone)DeleteNumberHandler( numId string , a *HandlersVars){
	NumID , _ := strconv.ParseInt(numId , 10 , 64)
	stmt , err :=a.db.Prepare("delete from Telephones where Numid=?")
	checkErr(err)
	_ ,err = stmt.Exec(NumID)
	checkErr(err)

}
func(tele Telephone)AddNumberHandler(newNumber string , contactId string, a *HandlersVars) (int64, int64){
	ContactID , _ := strconv.ParseInt(contactId , 10 , 64)
	id := addNum( newNumber, int(ContactID) , a)
	return ContactID, id


}
func addNum (number string , contactId int , a *HandlersVars)(int64){
	stmt2,err :=a.db.Prepare("INSERT Telephones SET number=? , contactid=?")
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
func serverContent ( urlParams map[string]string ,username string, a *HandlersVars)(page,string ,*template.Template ){
	staticPages := populateStaticPages()

	p := page{Contacts:[]ContactInfo{}}
	rows , err := a.db.Query("select * from Contacts where username =? " ,username)

	checkErr(err)

	var contact ContactInfo
	for rows.Next() {
		rows.Scan(&contact.Name ,&contact.Email ,&contact.Nationality ,&contact.Address , &contact.Username ,&contact.Id)
		p.Contacts = append(p.Contacts ,contact)
	}

	//mux.vars() -> creates a map of rout variables that can be retrieved
        //urlParams := mux.Vars(r)
	page_alias := urlParams["page_alias"]
	if page_alias ==""{
		page_alias="Home"
	}
	staticPage := staticPages.Lookup(page_alias+".html")
        return p,page_alias, staticPage


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