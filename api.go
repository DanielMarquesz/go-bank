package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleGetAccountByID), s.store))
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransferAccount))

	log.Println("JSON API server running on port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}

	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}

	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	return fmt.Errorf("Method not allowed %s", r.Method)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		fmt.Println("Mehotd not allowed!")
		return nil
	}

	req := new(LoginRequest)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, req)
}

// Adicionar Update Account
func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	// account := NewAccount("Daniel", "Marques")
	id, err := getID(r)
	if err != nil {
		return err
	}

	if r.Method == "GET" {

		fmt.Println("Getting account by id:", id)

		account, err := s.store.GetAccountByID(id)
		if err != nil {
			return err
		}
		return WriteJSON(w, http.StatusOK, account)
	}

	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	return WriteJSON(w, http.StatusOK, "Method not found")
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	req := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}

	account, err := NewAccount(req.FirstName, req.LastName, req.Password)
	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	tokenString, err := createJWT(account)
	if err != nil {
		return err
	}

	fmt.Println("TOKEN", tokenString)

	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {

	id, err := getID(r)
	if err != nil {
		return err
	}

	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, "Deleted")
}

func (s *APIServer) handleTransferAccount(w http.ResponseWriter, r *http.Request) error {

	transferReq := new(TransferRequest)
	if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
		return err
	}

	defer r.Body.Close()

	return WriteJSON(w, http.StatusOK, transferReq)
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()

	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func permissionDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, ApiError{Error: "permission denied"})
}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Calling JWT auth middleware")

		tokenString := r.Header.Get("x-jwt-token")
		token, err := validateJWT(tokenString)
		if err != nil {
			fmt.Println("Error validatin token", err)

			permissionDenied(w)
			return
		}

		if !token.Valid {
			fmt.Println("Invalid token")

			permissionDenied(w)
			return
		}

		userID, err := getID(r)
		if err != nil {

			fmt.Println("Error when getting userID")
			permissionDenied(w)
			return
		}
		account, err := s.GetAccountByID(userID)

		claims := token.Claims.(jwt.MapClaims)

		// See the type of a value
		// panic(reflect.TypeOf(claims["accountNumber"]))
		accountOfClaims := int64(claims["accountNumber"].(float64))

		if account.Number != accountOfClaims {

			fmt.Printf("The account number is different: accountNumber: %+v. claimAccount: %+v", account.Number, accountOfClaims)

			permissionDenied(w)
			return
		}

		if err != nil {
			WriteJSON(w, http.StatusForbidden, ApiError{Error: "Invalid token"})
		}

		handlerFunc(w, r)
	}
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Error string `json:"error"`
}

func validateJWT(tokenString string) (*jwt.Token, error) {

	fmt.Println("Validating JWT")

	secret := os.Getenv("JWT_SECRET")
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     15000,
		"accountNumber": account.Number,
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func getID(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]

	fmt.Println("Getting ID")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return id, fmt.Errorf("invalid id given %s", idStr)
	}
	return id, nil
}
