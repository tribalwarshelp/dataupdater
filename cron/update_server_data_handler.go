package cron

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/tribalwarshelp/shared/models"
	"golang.org/x/sync/errgroup"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
)

const (
	endpointConfig         = "/interface.php?func=get_config"
	endpointUnitConfig     = "/interface.php?func=get_unit_info"
	endpointBuildingConfig = "/interface.php?func=get_building_info"
	endpointPlayers        = "/map/player.txt.gz"
	endpointTribe          = "/map/ally.txt.gz"
	endpointVillage        = "/map/village.txt.gz"
	endpointKillAtt        = "/map/kill_att.txt.gz"
	endpointKillDef        = "/map/kill_def.txt.gz"
	endpointKillSup        = "/map/kill_sup.txt.gz"
	endpointKillAll        = "/map/kill_all.txt.gz"
	endpointKillAttTribe   = "/map/kill_att_tribe.txt.gz"
	endpointKillDefTribe   = "/map/kill_def_tribe.txt.gz"
	endpointKillAllTribe   = "/map/kill_all_tribe.txt.gz"
	endpointConquer        = "/map/conquer.txt.gz"
)

type updateServerDataHandler struct {
	baseURL      string
	db           *pg.DB
	server       *models.Server
	ennoblements []*models.Ennoblement
	pod          map[int]*models.OpponentsDefeated
	tod          map[int]*models.OpponentsDefeated
	players      []*models.Player
	tribes       []*models.Tribe
	villages     []*models.Village
}

type parsedODLine struct {
	ID    int
	Rank  int
	Score int
}

func (h *updateServerDataHandler) parseODLine(line []string) (*parsedODLine, error) {
	if len(line) != 3 {
		return nil, fmt.Errorf("Invalid line format (should be rank,id,score)")
	}
	p := &parsedODLine{}
	var err error
	p.Rank, err = strconv.Atoi(line[0])
	if err != nil {
		return nil, errors.Wrap(err, "parsedODLine.Rank")
	}
	p.ID, err = strconv.Atoi(line[1])
	if err != nil {
		return nil, errors.Wrap(err, "parsedODLine.ID")
	}
	p.Score, err = strconv.Atoi(line[2])
	if err != nil {
		return nil, errors.Wrap(err, "parsedODLine.Score")
	}
	return p, nil
}

func (h *updateServerDataHandler) getOD(tribe bool) (map[int]*models.OpponentsDefeated, error) {
	m := make(map[int]*models.OpponentsDefeated)
	urls := []string{
		fmt.Sprintf("%s%s", h.baseURL, endpointKillAll),
		fmt.Sprintf("%s%s", h.baseURL, endpointKillAtt),
		fmt.Sprintf("%s%s", h.baseURL, endpointKillDef),
		fmt.Sprintf("%s%s", h.baseURL, endpointKillSup),
	}
	if tribe {
		urls = []string{
			fmt.Sprintf("%s%s", h.baseURL, endpointKillAllTribe),
			fmt.Sprintf("%s%s", h.baseURL, endpointKillAttTribe),
			fmt.Sprintf("%s%s", h.baseURL, endpointKillDefTribe),
			"",
		}
	}
	for _, url := range urls {
		if url == "" {
			continue
		}
		lines, err := getCSVData(url, true)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get data, url %s", url)
		}
		for _, line := range lines {
			parsed, err := h.parseODLine(line)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to parse line, url %s", url)
			}
			if _, ok := m[parsed.ID]; !ok {
				m[parsed.ID] = &models.OpponentsDefeated{}
			}
			switch url {
			case urls[0]:
				m[parsed.ID].RankTotal = parsed.Rank
				m[parsed.ID].ScoreTotal = parsed.Score
			case urls[1]:
				m[parsed.ID].RankAtt = parsed.Rank
				m[parsed.ID].ScoreAtt = parsed.Score
			case urls[2]:
				m[parsed.ID].RankDef = parsed.Rank
				m[parsed.ID].ScoreDef = parsed.Score
			case urls[3]:
				m[parsed.ID].RankSup = parsed.Rank
				m[parsed.ID].ScoreSup = parsed.Score
			}
		}
	}
	return m, nil
}

func (h *updateServerDataHandler) parsePlayerLine(line []string) (*models.Player, error) {
	if len(line) != 6 {
		return nil, fmt.Errorf("Invalid line format (should be id,name,tribeid,villages,points,rank)")
	}

	var err error
	ex := true
	player := &models.Player{
		Exists: &ex,
	}
	player.ID, err = strconv.Atoi(line[0])
	if err != nil {
		return nil, errors.Wrap(err, "player.ID")
	}
	player.Name, err = url.QueryUnescape(line[1])
	if err != nil {
		return nil, errors.Wrap(err, "player.Name")
	}
	player.TribeID, err = strconv.Atoi(line[2])
	if err != nil {
		return nil, errors.Wrap(err, "player.TribeID")
	}
	player.TotalVillages, err = strconv.Atoi(line[3])
	if err != nil {
		return nil, errors.Wrap(err, "player.TotalVillages")
	}
	player.Points, err = strconv.Atoi(line[4])
	if err != nil {
		return nil, errors.Wrap(err, "player.Points")
	}
	player.Rank, err = strconv.Atoi(line[5])
	if err != nil {
		return nil, errors.Wrap(err, "player.Rank")
	}

	return player, nil
}

func (h *updateServerDataHandler) getPlayers(od map[int]*models.OpponentsDefeated,
	firstEnnoblementByID map[int]*models.Ennoblement) ([]*models.Player, error) {
	url := h.baseURL + endpointPlayers
	lines, err := getCSVData(url, true)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get data, url %s", url)
	}
	now := time.Now()

	players := []*models.Player{}
	for _, line := range lines {
		player, err := h.parsePlayerLine(line)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse line, url %s", url)
		}
		playerOD, ok := od[player.ID]
		if ok {
			player.OpponentsDefeated = *playerOD
		}
		firstEnnoblement, ok := firstEnnoblementByID[player.ID]
		if ok {
			diffInDays := getDateDifferenceInDays(now, firstEnnoblement.EnnobledAt)
			player.DailyGrowth = calcPlayerDailyGrowth(diffInDays, player.Points)
		}
		players = append(players, player)
	}

	return players, nil
}

func (h *updateServerDataHandler) parseTribeLine(line []string) (*models.Tribe, error) {
	if len(line) != 8 {
		return nil, fmt.Errorf("Invalid line format (should be id,name,tag,members,villages,points,allpoints,rank)")
	}

	var err error
	ex := true
	tribe := &models.Tribe{
		Exists: &ex,
	}
	tribe.ID, err = strconv.Atoi(line[0])
	if err != nil {
		return nil, errors.Wrap(err, "tribe.ID")
	}
	tribe.Name, err = url.QueryUnescape(line[1])
	if err != nil {
		return nil, errors.Wrap(err, "tribe.Name")
	}
	tribe.Tag, err = url.QueryUnescape(line[2])
	if err != nil {
		return nil, errors.Wrap(err, "tribe.Tag")
	}
	tribe.TotalMembers, err = strconv.Atoi(line[3])
	if err != nil {
		return nil, errors.Wrap(err, "tribe.TotalMembers")
	}
	tribe.TotalVillages, err = strconv.Atoi(line[4])
	if err != nil {
		return nil, errors.Wrap(err, "tribe.TotalVillages")
	}
	tribe.Points, err = strconv.Atoi(line[5])
	if err != nil {
		return nil, errors.Wrap(err, "tribe.Points")
	}
	tribe.AllPoints, err = strconv.Atoi(line[6])
	if err != nil {
		return nil, errors.Wrap(err, "tribe.AllPoints")
	}
	tribe.Rank, err = strconv.Atoi(line[7])
	if err != nil {
		return nil, errors.Wrap(err, "tribe.Rank")
	}

	return tribe, nil
}

func (h *updateServerDataHandler) getTribes(od map[int]*models.OpponentsDefeated, numberOfVillages int) ([]*models.Tribe, error) {
	url := h.baseURL + endpointTribe
	lines, err := getCSVData(url, true)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get data, url %s", url)
	}
	tribes := []*models.Tribe{}
	for _, line := range lines {
		tribe, err := h.parseTribeLine(line)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse line, url %s", url)
		}
		tribeOD, ok := od[tribe.ID]
		if ok {
			tribe.OpponentsDefeated = *tribeOD
		}
		if tribe.TotalVillages > 0 && numberOfVillages > 0 {
			tribe.Dominance = float64(tribe.TotalVillages) / float64(numberOfVillages) * 100
		} else {
			tribe.Dominance = 0
		}
		tribes = append(tribes, tribe)
	}
	return tribes, nil
}

func (h *updateServerDataHandler) parseVillageLine(line []string) (*models.Village, error) {
	if len(line) != 7 {
		return nil, fmt.Errorf("Invalid line format (should be id,name,x,y,playerID,points,bonus)")
	}
	var err error
	village := &models.Village{}
	village.ID, err = strconv.Atoi(line[0])
	if err != nil {
		return nil, errors.Wrap(err, "village.ID")
	}
	village.Name, err = url.QueryUnescape(line[1])
	if err != nil {
		return nil, errors.Wrap(err, "village.Name")
	}
	village.X, err = strconv.Atoi(line[2])
	if err != nil {
		return nil, errors.Wrap(err, "village.X")
	}
	village.Y, err = strconv.Atoi(line[3])
	if err != nil {
		return nil, errors.Wrap(err, "village.Y")
	}
	village.PlayerID, err = strconv.Atoi(line[4])
	if err != nil {
		return nil, errors.Wrap(err, "village.PlayerID")
	}
	village.Points, err = strconv.Atoi(line[5])
	if err != nil {
		return nil, errors.Wrap(err, "village.Points")
	}
	village.Bonus, err = strconv.Atoi(line[6])
	if err != nil {
		return nil, errors.Wrap(err, "village.Bonus")
	}
	return village, nil
}

func (h *updateServerDataHandler) getVillages() ([]*models.Village, error) {
	url := h.baseURL + endpointVillage
	lines, err := getCSVData(url, true)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get data, url %s", url)
	}
	villages := []*models.Village{}
	for _, line := range lines {
		village, err := h.parseVillageLine(line)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse line, url %s", url)
		}
		villages = append(villages, village)
	}
	return villages, nil
}

func (h *updateServerDataHandler) parseEnnoblementLine(line []string) (*models.Ennoblement, error) {
	if len(line) != 4 {
		return nil, fmt.Errorf("Invalid line format (should be village_id,timestamp,new_owner_id,old_owner_id)")
	}
	var err error
	ennoblement := &models.Ennoblement{}
	ennoblement.VillageID, err = strconv.Atoi(line[0])
	if err != nil {
		return nil, errors.Wrap(err, "ennoblement.VillageID")
	}
	timestamp, err := strconv.Atoi(line[1])
	if err != nil {
		return nil, errors.Wrap(err, "timestamp")
	}
	ennoblement.EnnobledAt = time.Unix(int64(timestamp), 0)
	ennoblement.NewOwnerID, err = strconv.Atoi(line[2])
	if err != nil {
		return nil, errors.Wrap(err, "ennoblement.NewOwnerID")
	}
	ennoblement.OldOwnerID, err = strconv.Atoi(line[3])
	if err != nil {
		return nil, errors.Wrap(err, "ennoblement.OldOwnerID")
	}

	return ennoblement, nil
}

func (h *updateServerDataHandler) getEnnoblements() ([]*models.Ennoblement, map[int]*models.Ennoblement, error) {
	url := h.baseURL + endpointConquer
	lines, err := getCSVData(url, true)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to get data, url %s", url)
	}

	lastEnnoblement := &models.Ennoblement{}
	if err := h.db.
		Model(lastEnnoblement).
		Limit(1).
		Order("ennobled_at DESC").
		Select(); err != nil && err != pg.ErrNoRows {
		return nil, nil, errors.Wrapf(err, "cannot load last ennoblement, url %s", url)
	}

	firstEnnoblementByID := make(map[int]*models.Ennoblement)
	ennoblements := []*models.Ennoblement{}
	for _, line := range lines {
		ennoblement, err := h.parseEnnoblementLine(line)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot parse line, url %s", url)
		}
		if otherEnnoblement, ok := firstEnnoblementByID[ennoblement.NewOwnerID]; !ok ||
			otherEnnoblement.EnnobledAt.After(ennoblement.EnnobledAt) {
			firstEnnoblementByID[ennoblement.NewOwnerID] = ennoblement
		}
		if ennoblement.EnnobledAt.After(lastEnnoblement.EnnobledAt) {
			ennoblements = append(ennoblements, ennoblement)
		}
	}
	return ennoblements, firstEnnoblementByID, nil
}

func (h *updateServerDataHandler) getConfig() (*models.ServerConfig, error) {
	url := h.baseURL + endpointConfig
	cfg := &models.ServerConfig{}
	err := getXML(url, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "getConfig")
	}
	return cfg, nil
}

func (h *updateServerDataHandler) getBuildingConfig() (*models.BuildingConfig, error) {
	url := h.baseURL + endpointBuildingConfig
	cfg := &models.BuildingConfig{}
	err := getXML(url, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "getBuildingConfig")
	}
	return cfg, nil
}

func (h *updateServerDataHandler) getUnitConfig() (*models.UnitConfig, error) {
	url := h.baseURL + endpointUnitConfig
	cfg := &models.UnitConfig{}
	err := getXML(url, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "getUnitConfig")
	}
	return cfg, nil
}

func (h *updateServerDataHandler) isTheSameAsServerHistoryUpdatedAt(t time.Time) bool {
	return t.Year() == h.server.HistoryUpdatedAt.Year() &&
		t.Month() == h.server.HistoryUpdatedAt.Month() &&
		t.Day() == h.server.HistoryUpdatedAt.Day()
}

func (h *updateServerDataHandler) calculateODifference(od1 models.OpponentsDefeated, od2 models.OpponentsDefeated) models.OpponentsDefeated {
	return models.OpponentsDefeated{
		RankAtt:    (od1.RankAtt - od2.RankAtt) * -1,
		ScoreAtt:   od1.ScoreAtt - od2.ScoreAtt,
		RankDef:    (od1.RankDef - od2.RankDef) * -1,
		ScoreDef:   od1.ScoreDef - od2.ScoreDef,
		RankSup:    (od1.RankSup - od2.RankSup) * -1,
		ScoreSup:   od1.ScoreSup - od2.ScoreSup,
		RankTotal:  (od1.RankTotal - od2.RankTotal) * -1,
		ScoreTotal: od1.ScoreTotal - od2.ScoreTotal,
	}
}

func (h *updateServerDataHandler) calculateDailyTribeStats(tribes []*models.Tribe,
	history []*models.TribeHistory) []*models.DailyTribeStats {
	dailyStats := []*models.DailyTribeStats{}
	searchableTribes := makeTribesSearchable(tribes)

	for _, historyRecord := range history {
		if !h.isTheSameAsServerHistoryUpdatedAt(historyRecord.CreateDate) {
			continue
		}
		if index := searchByID(searchableTribes, historyRecord.TribeID); index != 0 {
			tribe := tribes[index]
			dailyStats = append(dailyStats, &models.DailyTribeStats{
				TribeID:           tribe.ID,
				Members:           tribe.TotalMembers - historyRecord.TotalMembers,
				Villages:          tribe.TotalVillages - historyRecord.TotalVillages,
				Points:            tribe.Points - historyRecord.Points,
				AllPoints:         tribe.AllPoints - historyRecord.AllPoints,
				Rank:              (tribe.Rank - historyRecord.Rank) * -1,
				Dominance:         tribe.Dominance - historyRecord.Dominance,
				CreateDate:        historyRecord.CreateDate,
				OpponentsDefeated: h.calculateODifference(tribe.OpponentsDefeated, historyRecord.OpponentsDefeated),
			})
		}
	}

	return dailyStats
}

func (h *updateServerDataHandler) calculateDailyPlayerStats(players []*models.Player,
	history []*models.PlayerHistory) []*models.DailyPlayerStats {
	dailyStats := []*models.DailyPlayerStats{}
	searchablePlayers := makePlayersSearchable(players)

	for _, historyRecord := range history {
		if !h.isTheSameAsServerHistoryUpdatedAt(historyRecord.CreateDate) {
			continue
		}
		if index := searchByID(searchablePlayers, historyRecord.PlayerID); index != 0 {
			player := players[index]
			dailyStats = append(dailyStats, &models.DailyPlayerStats{
				PlayerID:          player.ID,
				Villages:          player.TotalVillages - historyRecord.TotalVillages,
				Points:            player.Points - historyRecord.Points,
				Rank:              (player.Rank - historyRecord.Rank) * -1,
				CreateDate:        historyRecord.CreateDate,
				OpponentsDefeated: h.calculateODifference(player.OpponentsDefeated, historyRecord.OpponentsDefeated),
			})
		}
	}

	return dailyStats
}

func (h *updateServerDataHandler) update() error {
	pod, err := h.getOD(false)
	if err != nil {
		return err
	}
	tod, err := h.getOD(true)
	if err != nil {
		return err
	}

	ennoblements, firstEnnoblementByID, err := h.getEnnoblements()
	if err != nil {
		return err
	}

	villages, err := h.getVillages()
	if err != nil {
		return err
	}
	numberOfVillages := len(villages)

	tribes, err := h.getTribes(tod, countPlayerVillages(villages))
	if err != nil {
		return err
	}
	numberOfTribes := len(tribes)

	players, err := h.getPlayers(pod, firstEnnoblementByID)
	if err != nil {
		return err
	}
	numberOfPlayers := len(players)

	cfg, err := h.getConfig()
	if err != nil {
		return err
	}
	buildingCfg, err := h.getBuildingConfig()
	if err != nil {
		return err
	}
	unitCfg, err := h.getUnitConfig()
	if err != nil {
		return err
	}

	errGroup, _ := errgroup.WithContext(context.Background())

	if len(tribes) > 0 {
		ids := []int{}
		for _, tribe := range tribes {
			ids = append(ids, tribe.ID)
		}
		errGroup.Go(func() error {
			tx, err := h.db.Begin()
			if err != nil {
				return err
			}
			defer tx.Close()
			if _, err := tx.Model(&tribes).
				OnConflict("(id) DO UPDATE").
				Set("name = EXCLUDED.name").
				Set("tag = EXCLUDED.tag").
				Set("total_members = EXCLUDED.total_members").
				Set("total_villages = EXCLUDED.total_villages").
				Set("points = EXCLUDED.points").
				Set("all_points = EXCLUDED.all_points").
				Set("rank = EXCLUDED.rank").
				Set("exists = EXCLUDED.exists").
				Set("dominance = EXCLUDED.dominance").
				Apply(attachODSetClauses).
				Insert(); err != nil {
				return errors.Wrap(err, "cannot insert tribes")
			}
			if _, err := tx.Model(&tribes).
				Where("tribe.id NOT IN (?)", pg.In(ids)).
				Set("exists = false").
				Update(); err != nil && err != pg.ErrNoRows {
				return errors.Wrap(err, "cannot update not exist tribes")
			}
			return tx.Commit()
		})

		errGroup.Go(func() error {
			tribesHistory := []*models.TribeHistory{}
			if err := h.db.Model(&tribesHistory).
				DistinctOn("tribe_id").
				Column("*").
				Where("tribe_id IN (?)", pg.In(ids)).
				Order("tribe_id DESC", "create_date DESC").
				Select(); err != nil && err != pg.ErrNoRows {
				return errors.Wrap(err, "cannot select tribes history")
			}
			todaysTribesStats := h.calculateDailyTribeStats(tribes, tribesHistory)
			if len(todaysTribesStats) > 0 {
				if _, err := h.db.
					Model(&todaysTribesStats).
					OnConflict("ON CONSTRAINT daily_tribe_stats_tribe_id_create_date_key DO UPDATE").
					Set("members = EXCLUDED.members").
					Set("villages = EXCLUDED.villages").
					Set("points = EXCLUDED.points").
					Set("all_points = EXCLUDED.all_points").
					Set("rank = EXCLUDED.rank").
					Set("dominance = EXCLUDED.dominance").
					Apply(attachODSetClauses).
					Insert(); err != nil {
					return errors.Wrap(err, "cannot insert today's tribes stats")
				}
			}
			return nil
		})
	}
	if len(players) > 0 {
		ids := []int{}
		for _, player := range players {
			ids = append(ids, player.ID)
		}
		errGroup.Go(func() error {
			t := time.Now()
			tx, err := h.db.Begin()
			if err != nil {
				return err
			}
			defer tx.Close()
			if _, err := tx.Model(&players).
				OnConflict("(id) DO UPDATE").
				Set("name = EXCLUDED.name").
				Set("total_villages = EXCLUDED.total_villages").
				Set("points = EXCLUDED.points").
				Set("rank = EXCLUDED.rank").
				Set("exists = EXCLUDED.exists").
				Set("tribe_id = EXCLUDED.tribe_id").
				Set("daily_growth = EXCLUDED.daily_growth").
				Apply(attachODSetClauses).
				Insert(); err != nil {
				return errors.Wrap(err, "cannot insert players")
			}
			if _, err := tx.Model(&models.Player{}).
				Where("id NOT IN (?)", pg.In(ids)).
				Set("exists = false").
				Update(); err != nil && err != pg.ErrNoRows {
				return errors.Wrap(err, "cannot update not exist players")
			}
			log.Println("players", time.Since(t))
			return tx.Commit()
		})

		errGroup.Go(func() error {
			tx, err := h.db.Begin()
			if err != nil {
				return err
			}
			defer tx.Close()
			playerHistory := []*models.PlayerHistory{}
			if err := h.db.Model(&playerHistory).
				DistinctOn("player_id").
				Column("*").
				Where("player_id IN (?)", pg.In(ids)).
				Order("player_id DESC", "create_date DESC").Select(); err != nil && err != pg.ErrNoRows {
				return errors.Wrap(err, "cannot select players history")
			}
			todaysPlayersStats := h.calculateDailyPlayerStats(players, playerHistory)
			if len(todaysPlayersStats) > 0 {
				if _, err := tx.
					Model(&todaysPlayersStats).
					OnConflict("ON CONSTRAINT daily_player_stats_player_id_create_date_key DO UPDATE").
					Set("villages = EXCLUDED.villages").
					Set("points = EXCLUDED.points").
					Set("rank = EXCLUDED.rank").
					Apply(attachODSetClauses).
					Insert(); err != nil {
					return errors.Wrap(err, "cannot insert today's players stats")
				}
			}
			return tx.Commit()
		})
	}
	if len(villages) > 0 {
		errGroup.Go(func() error {
			tx, err := h.db.Begin()
			if err != nil {
				return err
			}
			defer tx.Close()
			if _, err := tx.Model(&models.Village{}).
				Where("true").
				Delete(); err != nil && err != pg.ErrNoRows {
				return errors.Wrap(err, "cannot delete villages")
			}
			if _, err := tx.Model(&villages).
				OnConflict("(id) DO UPDATE").
				Set("name = EXCLUDED.name").
				Set("points = EXCLUDED.points").
				Set("x = EXCLUDED.x").
				Set("y = EXCLUDED.y").
				Set("bonus = EXCLUDED.bonus").
				Set("player_id = EXCLUDED.player_id").
				Insert(); err != nil {
				return errors.Wrap(err, "cannot insert villages")
			}
			return tx.Commit()
		})
	}
	if len(ennoblements) > 0 {
		errGroup.Go(func() error {
			if _, err := h.db.Model(&ennoblements).Insert(); err != nil {
				return errors.Wrap(err, "cannot insert ennoblements")
			}
			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		return err
	}

	if _, err := h.db.Model(h.server).
		Set("data_updated_at = ?", time.Now()).
		Set("unit_config = ?", unitCfg).
		Set("building_config = ?", buildingCfg).
		Set("config = ?", cfg).
		Set("number_of_players = ?", numberOfPlayers).
		Set("number_of_tribes = ?", numberOfTribes).
		Set("number_of_villages = ?", numberOfVillages).
		Returning("*").
		WherePK().
		Update(); err != nil {
		return errors.Wrap(err, "cannot update server")
	}

	return nil
}

func attachODSetClauses(q *orm.Query) (*orm.Query, error) {
	return q.Set("rank_att = EXCLUDED.rank_att").
			Set("score_att = EXCLUDED.score_att").
			Set("rank_def = EXCLUDED.rank_def").
			Set("score_def = EXCLUDED.score_def").
			Set("rank_sup = EXCLUDED.rank_sup").
			Set("score_sup = EXCLUDED.score_sup").
			Set("rank_total = EXCLUDED.rank_total").
			Set("score_total = EXCLUDED.score_total"),
		nil
}
