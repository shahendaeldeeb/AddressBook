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
	Id 		int      ` db:"id"`
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
func (middleware middlewareHandlers) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc){
	// Updated to pass app.handlervars as a parameter to our handler type.
	middleware.NewMiddleWarHandler(w, r, next, middleware.HandlersVars)

}


func main() {
	muxer := mux.NewRouter()
	dataBase := initDb()
	defer dataBase.Close()
	usedDataBase := &HandlersVars{db: dataBase}

	muxer.HandleFunc("/viewnumbers/{id}", func(w http.ResponseWriter, r *http.Request) {
		var tele Telephone
		if validateId(mux.Vars(r)["id"]) {
			numbers := tele.View(mux.Vars(r)["id"], usedDataBase)
			encoder := json.NewEncoder(w)
			err := encoder.Encode(numbers)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
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

		id := contact.Save(usedDataBase)
                contact.Id = int(id)
		_ = contact.addNum(contact.Number, usedDataBase)
		sessions.GetSession(r).Set("Contact", id)
	})
	muxer.HandleFunc("/contact/{id}", func(w http.ResponseWriter, r *http.Request) {
		var contact ContactInfo
		if validateId(mux.Vars(r)["id"]) {
			contact.Delete(mux.Vars(r)["id"], usedDataBase)
		}
	}).Methods("DELETE")
	muxer.HandleFunc("/deletenumber/{numId}", func(w http.ResponseWriter, r *http.Request) {
		var telephone Telephone
		if validateId(mux.Vars(r)["numId"]) {
			telephone.Delete(mux.Vars(r)["numId"], usedDataBase)
		}
	}).Methods("DELETE")
	muxer.HandleFunc("/addnumber/{ContactId}", func(w http.ResponseWriter, r *http.Request) {
		var telephone Telephone
		if errs := validator.Validate(telephone); errs != nil {
			checkErr(errs)
		}
		telephone.Number = r.FormValue("NewNumber")
		if validateId(mux.Vars(r)["ContactId"]) {
			ContactID, id := telephone.Add(mux.Vars(r)["ContactId"], usedDataBase)

			telephone.Num_id = int(id)
			telephone.ContactId = int(ContactID)
			encoder := json.NewEncoder(w)
			err := encoder.Encode(telephone)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}).Methods("POST")
	muxer.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		sessions.GetSession(r).Set("User", "")
		http.Redirect(w, r, "/login", http.StatusFound)
	})
	muxer.HandleFunc("/", func(w http.ResponseWriter, r*http.Request){
		var contact ContactInfo
		contact.Username = sessions.GetSession(r).Get("User").(string)
		page_alias, staticPage := serverContent(mux.Vars(r), usedDataBase)
		if page_alias == "Home"{
			p := page{Contacts:contact.display(usedDataBase)}
			staticPage.Execute(w, p)
		}else{
			staticPage.Execute(w, nil)
		}
	})
	muxer.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		var p LoginPage
		info := User{Username:r.FormValue("username"), Password:[]byte(r.FormValue("password"))}
		if errs := validator.Validate(info); errs != nil {
			checkErr(errs)
		}
		if r.FormValue("signUp") != "" {
			if info.exists(usedDataBase) {
				p.Error = "User already exist"
			} else {
				info.signUp(usedDataBase)
				sessions.GetSession(r).Set("User", r.FormValue("username"))
				http.Redirect(w, r, "/Home", http.StatusFound)
				return
			}
		} else if r.FormValue("login") != "" {
			if info.login(usedDataBase) {
				sessions.GetSession(r).Set("User", r.FormValue("username"))
				http.Redirect(w, r, "/Home", http.StatusFound)
				return
			} else {
				p.Error = "Failed login"
				return
			}
		}



		templates := template.Must(template.ParseFiles("Login.html"))
		err := templates.Execute(w, p)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}).Methods("GET")
	muxer.HandleFunc("/{page_alias}", func(w http.ResponseWriter ,r*http.Request){
		var contact ContactInfo
		contact.Username = sessions.GetSession(r).Get("User").(string)
		page_alias, staticPage := serverContent(mux.Vars(r), usedDataBase)
		if page_alias == "Home"{
			p := page{Contacts:contact.display(usedDataBase)}
			staticPage.Execute(w, p)
		}else{
			staticPage.Execute(w, nil)
		}
	})
	muxer.HandleFunc("/img/", serverResource)
	muxer.HandleFunc("/js/", serverResource)
	muxer.HandleFunc("/css/{page_alias}", serverResource)


	//it provides some default middleware
	n := negroni.Classic()
	n.Use(sessions.Sessions("go-for-web-dev", cookiestore.New([]byte("my-secret-123"))))
	//add Handler to middleware stack
	n.Use(middlewareHandlers{usedDataBase, verifyDatabase})
	// to add http.Handler (process that runs in response to request made to web app.)alli f el mux in negroni stack
	n.UseHandler(muxer)
	n.Run(":8080")

}
func validateId(id string) bool{
	if id != ""{
		return true
	}else {
		return false
	}
}
func initDb() *sql.DB{
	db, err := sql.Open("mysql", "root:shahenda_hassan@/mydatabase")
	if err != nil {
		panic(err.Error())
	}
	return db

}
func verifyDatabase(w http.ResponseWriter, r *http.Request, next http.HandlerFunc, a *HandlersVars){
	//ping() -> it verify connection to database is still alive , establishing connection if necessary
	err := a.db.Ping();
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	next(w, r)
}
func (info User) login(a *HandlersVars) bool {
	row, err := a.db.Query("SELECT userpassword FROM Users WHERE username =?", info.Username)
	defer row.Close()
	checkErr(err)
	var dbPasswordHash []byte
	if row.Next() {
		row.Scan(&dbPasswordHash)
	}
	err = bcrypt.CompareHashAndPassword(dbPasswordHash, info.Password)
	return err == nil
}
func (info User) exists(a *HandlersVars) bool {
	row, err := a.db.Query("SELECT * FROM Users WHERE username =?", info.Username )
	defer row.Close()
	checkErr(err)
	return row.Next()
}
func (info User) signUp(a *HandlersVars) {
	var err error
	secret, err := bcrypt.GenerateFromPassword(info.Password, bcrypt.DefaultCost)
	checkErr(err)
	stmt, err := a.db.Prepare("INSERT Users SET username=?, userpassword=?")
	defer stmt.Close()
	checkErr(err)
	_, err = stmt.Exec(info.Username, secret)
}
func (contact ContactInfo) display(a *HandlersVars)[]ContactInfo{
	var Contacts []ContactInfo
	rows, err := a.db.Query("select * from Contacts where username =? ", contact.Username)
	checkErr(err)
	for rows.Next() {
		rows.Scan(&contact.Name,  &contact.Email, &contact.Nationality, &contact.Address, &contact.Username, &contact.Id)
		Contacts = append(Contacts, contact)
	}
	return Contacts
}
func (contactInfo ContactInfo) Save(a *HandlersVars) int64 {
	stmt, err := a.db.Prepare("INSERT Contacts SET name=?, email=?, nationality=?, address=?, username=?")
	checkErr(err)
	res, err := stmt.Exec(contactInfo.Name, contactInfo.Email, contactInfo.Nationality, contactInfo.Address, contactInfo.Username)
	checkErr(err)
	id, err := res.LastInsertId()
	checkErr(err)
	return id
}
func (contact ContactInfo) Delete(id string, a *HandlersVars){
	ID, _ := strconv.ParseInt(id, 10, 64)
	stmt, err := a.db.Prepare("delete from Contacts where id=?")
	checkErr(err)
	_, err = stmt.Exec(ID)
	checkErr(err)

}
func (tele Telephone ) View(id string, a *HandlersVars) []Telephone {
	ID, _ := strconv.ParseInt(id, 10, 64)
	numbers := []Telephone{}
	rows, err := a.db.Query("select * from Telephones where contactid =?", ID)
	checkErr(err)
	for rows.Next() {
		rows.Scan(&tele.ContactId, &tele.Number, &tele.Num_id)
		numbers = append(numbers, Telephone{ContactId:tele.ContactId, Number:tele.Number, Num_id:tele.Num_id })
	}
	return numbers

}
func (tele Telephone) Delete(numId string, a *HandlersVars){
	NumID, _ := strconv.ParseInt(numId, 10, 64)
	stmt, err := a.db.Prepare("delete from Telephones where Numid=?")
	checkErr(err)
	_, err = stmt.Exec(NumID)
	checkErr(err)

}
func (tele Telephone) Add(contactId string, a *HandlersVars)(int64, int64) {
	contactID, _ := strconv.ParseInt(contactId, 10, 64)
	contactInfo := ContactInfo{Id: int(contactID)}
	id := contactInfo.addNum(tele.Number, a)
	return contactID, id
}
func (contactInfo ContactInfo) addNum(number string, a *HandlersVars) int64 {
	stmt2, err := a.db.Prepare("INSERT Telephones SET number=?, contactid=?")
	checkErr(err)
	stmt, err := stmt2.Exec(number, contactInfo.Id)
	checkErr(err)
	id, err :=stmt.LastInsertId()
	checkErr(err)
	return id
}

//-------------------------------Static Pages Handle Functions ---------------------------------------------
//retrieve all static pages
func populateStaticPages() *template.Template{
	result := template.New("templates")
	templatePathes := new([]string)
	basepath := "pages"
	templateFolder, _ := os.Open(basepath)
	defer templateFolder.Close()
	//READdir-> to read directory name and return a list of directory entries
	// it takes -1 to return all files in single slice
	templatePathRaw, _ := templateFolder.Readdir(-1)
	for _, pathinfo := range templatePathRaw {
		log.Println(pathinfo.Name())
		*templatePathes = append(*templatePathes, basepath+"/"+pathinfo.Name())
	}
	//parsefile-> it parses source code and return corresponding file node
	result.ParseFiles(*templatePathes...)
	return  result
}
func serverContent(urlParams map[string]string, a *HandlersVars)(string, *template.Template ){
	staticPages := populateStaticPages()
	page_alias := urlParams["page_alias"]
	if page_alias == ""{
		page_alias = "Home"
	}
	staticPage := staticPages.Lookup(page_alias+".html")
        return page_alias, staticPage
}
// it serve file of type img , js , css
func serverResource(w http.ResponseWriter, r *http.Request){
	path := "public/bs4"  + r.URL.Path
	var contentType string
	if strings.HasSuffix(path, ".css"){
		contentType = "text/css; char-set=utf-8"
	}else if strings.HasSuffix(path, ".png"){
		contentType = "image/png; char-set=utf-8"
	}else if strings.HasSuffix(path, ".jpg"){
		contentType = "image/jpg; char-set=utf-8"
	}else if strings.HasSuffix(path, ".js"){
		contentType = "application/javascript; char-set=utf-8"
	}else {
		contentType = "text/plain; char-set=utf-8"
	}
	//log.Println(path)
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		w.Header().Add("content-type", contentType)
		br := bufio.NewReader(f)
		br.WriteTo(w)
	}else {
		w.WriteHeader(404)
	}
}
func checkErr(err error){
	if err != nil {
		panic(err)
	}
}