package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Itemm struct {
	Id           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

type Relation struct {
	Index []struct {
		Id             int                 `json:"id"`
		DatesLocations map[string][]string `json:"datesLocations"`
	} `json:"index"`
}

// Structure pour stocker les variables de la page
type PageVariables struct {
	FilteredArtists map[int]struct {
		Itemm
		DatesLocations map[string][]string
	}
	SearchTerm      string
	SearchType      string
	MinCreationDate string
	MaxCreationDate string
	MinAlbumDate    string
	MaxAlbumDate    string
	NumMembers      string
}

// Ajouter une nouvelle constante pour les critères de recherche
const (
	SearchByName         = "name"
	SearchByFirstAlbum   = "firstalbum"
	SearchByMembers      = "members"
	SearchByCreationDate = "creationdate"
	SearchByLocations    = "locations"
)

var idInfoMap map[int]struct {
	Itemm
	DatesLocations map[string][]string
}

func main() {
	apiURL := "https://groupietrackers.herokuapp.com/api/artists"
	apiURL2 := "https://groupietrackers.herokuapp.com/api/relation"

	// Première requête HTTP
	response, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Erreur lors de la requête HTTP:", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Println("La requête a retourné un code de statut non 200 OK:", response.StatusCode)
		return
	}

	var items []Itemm
	err = json.NewDecoder(response.Body).Decode(&items)
	if err != nil {
		fmt.Println("Erreur lors du décodage JSON:", err)
		return
	}

	// Deuxième requête HTTP
	response2, err := http.Get(apiURL2)
	if err != nil {
		fmt.Println("Erreur lors de la requête HTTP:", err)
		return
	}
	defer response2.Body.Close()

	if response2.StatusCode != http.StatusOK {
		fmt.Println("La requête a retourné un code de statut non 200 OK:", response2.StatusCode)
		return
	}

	var itemR Relation
	err = json.NewDecoder(response2.Body).Decode(&itemR)
	if err != nil {
		fmt.Println("Erreur lors du décodage JSON:", err)
		return
	}

	// Initialiser la carte idInfoMap
	idInfoMap = make(map[int]struct {
		Itemm
		DatesLocations map[string][]string
	})

	// Regrouper les informations par ID à partir de la première requête
	for _, item := range items {
		idInfoMap[item.Id] = struct {
			Itemm
			DatesLocations map[string][]string
		}{item, nil}
	}

	// Mettre à jour la carte avec les informations de la deuxième requête
	for _, item := range itemR.Index {
		if existingInfo, ok := idInfoMap[item.Id]; ok {
			existingInfo.DatesLocations = item.DatesLocations
			idInfoMap[item.Id] = existingInfo
		} else {
			// Si l'ID n'existe pas encore dans la carte, ajoutez-le
			idInfoMap[item.Id] = struct {
				Itemm
				DatesLocations map[string][]string
			}{Itemm{Id: item.Id}, item.DatesLocations}
		}
	}

	// Gérer les requêtes HTTP
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", handleindex)

	// Démarrer le serveur HTTP sur le port 8080
	port := 8080
	fmt.Printf("Serveur écoutant sur le port %d...\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Erreur lors du démarrage du serveur:", err)
	}
}

func handleindex(w http.ResponseWriter, r *http.Request) {
	// Extraire le terme de recherche et le type de recherche de la requête
	searchTerm := r.URL.Query().Get("search")
	searchType := r.URL.Query().Get("searchType")
	minCreationDate := r.URL.Query().Get("minCreationDate")
	maxCreationDate := r.URL.Query().Get("maxCreationDate")
	minAlbumDate := r.URL.Query().Get("minFirstAlbum")
	maxAlbumDate := r.URL.Query().Get("maxFirstAlbum")
	numMembersValues := r.URL.Query().Get("NumMembers")

	// Filtrer les artistes en fonction du terme de recherche et du type de recherche
	filteredArtists := filterArtists(idInfoMap, searchTerm, searchType, minCreationDate, maxCreationDate, minAlbumDate, maxAlbumDate, numMembersValues)

	// Rendre la page avec les résultats filtrés
	renderTemplate(w, filteredArtists, searchTerm, searchType, minCreationDate, maxCreationDate, minAlbumDate, maxAlbumDate, numMembersValues)
}

// Fonction pour filtrer les artistes en fonction du terme de recherche et du type de recherche
func filterArtists(idInfoMap map[int]struct {
	Itemm
	DatesLocations map[string][]string
}, searchTerm, searchType string, minCreationDate string, maxCreationDate string, minAlbumDate string, maxAlbumDate string, numMembersValues string) map[int]struct {
	Itemm
	DatesLocations map[string][]string
} {
	filteredArtists := make(map[int]struct {
		Itemm
		DatesLocations map[string][]string
	})

	minCreation, _ := strconv.Atoi(minCreationDate)
	maxCreation, _ := strconv.Atoi(maxCreationDate)
	minAlbum, _ := strconv.Atoi(minAlbumDate)
	maxAlbum, _ := strconv.Atoi(maxAlbumDate)

	for id, info := range idInfoMap {
		passesFilters := true // Initialise une variable pour suivre si l'artiste passe les filtres ou non

		parsedate, err := time.Parse("02-01-2006", info.FirstAlbum)
		year := parsedate.Year()
		if err != nil {
			fmt.Println("Erreur lors de la conversion de la date:", err)
		}

		// Filtrer les artistes en fonction du nombre de membres, si une valeur est spécifiée
		if numMembersValues != "" && !numMembersMatches(info.Members, numMembersValues) {
			passesFilters = false
		}

		// Filtrer les artistes en fonction de la date de création de l'album, si les dates minimale et maximale sont spécifiées
		if minAlbumDate != "" && maxAlbumDate != "" && (year < minAlbum || year > maxAlbum) {
			passesFilters = false
		}

		// Filtrer les artistes en fonction de la date de création du groupe, si les dates minimale et maximale sont spécifiées
		if minCreationDate != "" && maxCreationDate != "" && (info.CreationDate < minCreation || info.CreationDate > maxCreation) {
			passesFilters = false
		}

		// Filtrer les artistes en fonction du type de recherche, si un type est spécifié
		if searchType != "" {
			switch searchType {
			case SearchByName:
				if !strings.Contains(strings.ToLower(info.Name), strings.ToLower(searchTerm)) {
					passesFilters = false
				}
			case SearchByFirstAlbum:
				if !strings.Contains(strings.ToLower(info.FirstAlbum), strings.ToLower(searchTerm)) {
					passesFilters = false
				}
			case SearchByMembers:
				if !strings.Contains(strings.ToLower(strings.Join(info.Members, ",")), strings.ToLower(searchTerm)) {
					passesFilters = false
				}
			case SearchByCreationDate:
				if !strings.Contains(strings.ToLower(strconv.Itoa(info.CreationDate)), strings.ToLower(searchTerm)) {
					passesFilters = false
				}
			case SearchByLocations:
				locationFound := false
				for location, dates := range info.DatesLocations {
					for _, date := range dates {
						if strings.Contains(strings.ToLower(location), strings.ToLower(searchTerm)) || strings.Contains(strings.ToLower(date), strings.ToLower(searchTerm)) {
							locationFound = true
							break
						}
					}
					if locationFound {
						break
					}
				}
				if !locationFound {
					passesFilters = false
				}
			}
		}

		// Si aucun filtre spécifique n'est passé, mais un terme de recherche est spécifié, chercher par nom d'artiste ou membres
		if searchType == "" && searchTerm != "" {
			if !strings.Contains(strings.ToLower(info.Name), strings.ToLower(searchTerm)) && !strings.Contains(strings.ToLower(strings.Join(info.Members, ",")), strings.ToLower(searchTerm)) {
				passesFilters = false
			}
		}

		// Si l'artiste passe tous les filtres, l'ajouter à la liste filtrée
		if passesFilters {
			filteredArtists[id] = info
		}
	}

	return filteredArtists

}

func numMembersMatches(members []string, value string) bool {
	if value == "" {
		// Si la valeur est une chaîne vide, retourner false directement
		return false
	}
	numMembers, err := strconv.Atoi(value)
	if err != nil {
		return false // Le terme de recherche n'est pas un nombre valide
	}
	return len(members) == numMembers
}

// Fonction pour rendre le modèle HTML avec les données
func renderTemplate(w http.ResponseWriter, filteredArtists map[int]struct {
	Itemm
	DatesLocations map[string][]string
}, searchTerm, searchType string, minCreationDate, maxCreationDate string, minAlbumDate, maxAlbumDate string, nummembers string) {
	// Charger le modèle HTML
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Erreur lors du rendu de la page"+err.Error(), http.StatusInternalServerError)
		return
	}

	// Vérifier s'il y a un seul résultat
	if len(filteredArtists) == 1 {
		// Si un seul résultat est trouvé, afficher les détails de l'artiste comme actuellement
		for _, artist := range filteredArtists {
			err = tmpl.Execute(w, PageVariables{
				FilteredArtists: map[int]struct {
					Itemm
					DatesLocations map[string][]string
				}{1: artist},
				SearchTerm:      searchTerm,
				SearchType:      searchType,
				MinCreationDate: minCreationDate,
				MaxCreationDate: maxCreationDate,
				MinAlbumDate:    minAlbumDate,
				MaxAlbumDate:    maxAlbumDate,
				NumMembers:      nummembers,
			})
			if err != nil {
				http.Error(w, "Erreur lors de l'exécution du modèle HTML"+err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}

	// Si plusieurs résultats sont trouvés, afficher uniquement les images des artistes
	err = tmpl.Execute(w, PageVariables{
		FilteredArtists: filteredArtists,
	})
	if err != nil {
		http.Error(w, "Erreur lors de l'exécution du modèle HTML"+err.Error(), http.StatusInternalServerError)
		return
	}
}
