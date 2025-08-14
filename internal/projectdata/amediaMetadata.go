package projectdata

// AmediaProjectMetadata - Represents Amedia Metadata format
type AmediaProjectMetadata struct {
	Movies []struct {
		Actors         string `json:"actors"`
		AgeRestriction string `json:"age_restriction"`
		CmsID          int64  `json:"cms_id"`
		Country        string `json:"country"`
		Directors      string `json:"directors"`
		EndDate        string `json:"end_date"`
		File           struct {
			Duration *float64 `json:"duration"`
			Serid    *string  `json:"serid"`
		} `json:"file"`
		Genre          string `json:"genre"`
		GUID           string `json:"guid"`
		ImdbID         string `json:"imdb_id"`
		KinopoiskID    string `json:"kinopoisk_id"`
		OriginalTitle  string `json:"original_title"`
		Quote          string `json:"quote"`
		QuoteAuthor    string `json:"quote_author"`
		RusDescription string `json:"rus_description"`
		RusTitle       string `json:"rus_title"`
		StartDate      string `json:"start_date"`
		Year           int64  `json:"year"`
	} `json:"movies"`
	Series []struct {
		AgeRestriction      string `json:"age_restriction"`
		CmsID               int64  `json:"cms_id"`
		Country             string `json:"country"`
		EndDate             string `json:"end_date"`
		Genre               string `json:"genre"`
		GUID                string `json:"guid"`
		ImdbID              string `json:"imdb_id"`
		KinopoiskID         string `json:"kinopoisk_id"`
		OriginalBroadcaster string `json:"original_broadcaster"`
		OriginalTitle       string `json:"original_title"`
		Quote               string `json:"quote"`
		QuoteAuthor         string `json:"quote_author"`
		RusDescription      string `json:"rus_description"`
		RusTitle            string `json:"rus_title"`
		Seasons             []struct {
			Actors    string `json:"actors"`
			CmsID     int64  `json:"cms_id"`
			Directors string `json:"directors"`
			EndDate   string `json:"end_date"`
			Episodes  []struct {
				CmsID           int64  `json:"cms_id"`
				EndDate         string `json:"end_date"`
				EpisodeSynopsis string `json:"episode_synopsis"`
				File            struct {
					Duration *float64 `json:"duration"`
					Serid    *string  `json:"serid"`
				} `json:"file"`
				GUID                string `json:"guid"`
				OrderNumber         int64  `json:"order_number"`
				OriginalEpisodeName string `json:"original_episode_name"`
				RusEpisodeName      string `json:"rus_episode_name"`
				StartDate           string `json:"start_date"`
				Year                int64  `json:"year"`
			} `json:"episodes"`
			GUID              string `json:"guid"`
			OrderNumber       int64  `json:"order_number"`
			OrigName          string `json:"orig_name"`
			RusName           string `json:"rus_name"`
			SeasonDescription string `json:"season_description"`
			StartDate         string `json:"start_date"`
			YearsI            int64  `json:"years,omitempty"`
			YearsS            string `json:"years,omitempty"`
		} `json:"seasons"`
		StartDate string `json:"start_date"`
		YearsI    int64  `json:"years,omitempty"`
		YearsS    string `json:"years,omitempty"`
	} `json:"series"`
}
