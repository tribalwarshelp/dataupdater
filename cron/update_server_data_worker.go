package cron

import (
	"time"

	"github.com/tribalwarshelp/shared/models"
	"github.com/tribalwarshelp/shared/tw/dataloader"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
)

type updateServerDataWorker struct {
	db         *pg.DB
	dataloader dataloader.DataLoader
	server     *models.Server
}

func (h *updateServerDataWorker) loadPlayers(od map[int]*models.OpponentsDefeated) ([]*models.Player, error) {
	ennoblements := []*models.Ennoblement{}
	err := h.db.Model(&ennoblements).DistinctOn("new_owner_id").Order("new_owner_id ASC", "ennobled_at ASC").Select()
	if err != nil {
		return nil, errors.Wrap(err, "loadPlayers: Couldn't load ennoblements")
	}

	players, err := h.dataloader.LoadPlayers()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	searchableByNewOwnerID := &ennoblementsSearchableByNewOwnerID{ennoblements: ennoblements}
	for _, player := range players {
		playerOD, ok := od[player.ID]
		if ok {
			player.OpponentsDefeated = *playerOD
		}
		firstEnnoblementIndex := searchByID(searchableByNewOwnerID, player.ID)
		if firstEnnoblementIndex != -1 {
			firstEnnoblement := ennoblements[firstEnnoblementIndex]
			diffInDays := getDateDifferenceInDays(now, firstEnnoblement.EnnobledAt)
			player.DailyGrowth = calcPlayerDailyGrowth(diffInDays, player.Points)
		}
	}
	return players, nil
}

func (h *updateServerDataWorker) loadTribes(od map[int]*models.OpponentsDefeated, numberOfVillages int) ([]*models.Tribe, error) {
	tribes, err := h.dataloader.LoadTribes()
	if err != nil {
		return nil, err
	}
	for _, tribe := range tribes {
		tribeOD, ok := od[tribe.ID]
		if ok {
			tribe.OpponentsDefeated = *tribeOD
		}
		if tribe.TotalVillages > 0 && numberOfVillages > 0 {
			tribe.Dominance = float64(tribe.TotalVillages) / float64(numberOfVillages) * 100
		} else {
			tribe.Dominance = 0
		}
	}
	return tribes, nil
}

func (h *updateServerDataWorker) loadEnnoblements() ([]*models.Ennoblement, error) {
	lastEnnoblement := &models.Ennoblement{}
	if err := h.db.
		Model(lastEnnoblement).
		Limit(1).
		Order("ennobled_at DESC").
		Select(); err != nil && err != pg.ErrNoRows {
		return nil, errors.Wrapf(err, "couldnt load last ennoblement")
	}

	return h.dataloader.LoadEnnoblements(&dataloader.LoadEnnoblementsConfig{
		EnnobledAtGTE: lastEnnoblement.EnnobledAt,
	})
}

func (h *updateServerDataWorker) isTheSameAsServerHistoryUpdatedAt(t time.Time) bool {
	return t.Year() == h.server.HistoryUpdatedAt.Year() &&
		t.Month() == h.server.HistoryUpdatedAt.Month() &&
		t.Day() == h.server.HistoryUpdatedAt.Day()
}

func (h *updateServerDataWorker) calculateODifference(od1 models.OpponentsDefeated, od2 models.OpponentsDefeated) models.OpponentsDefeated {
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

func (h *updateServerDataWorker) calculateTodaysTribeStats(tribes []*models.Tribe,
	history []*models.TribeHistory) []*models.DailyTribeStats {
	todaysStats := []*models.DailyTribeStats{}
	searchableTribes := makeTribesSearchable(tribes)

	for _, historyRecord := range history {
		if !h.isTheSameAsServerHistoryUpdatedAt(historyRecord.CreateDate) {
			continue
		}
		if index := searchByID(searchableTribes, historyRecord.TribeID); index != -1 {
			tribe := tribes[index]
			todaysStats = append(todaysStats, &models.DailyTribeStats{
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

	return todaysStats
}

func (h *updateServerDataWorker) calculateDailyPlayerStats(players []*models.Player,
	history []*models.PlayerHistory) []*models.DailyPlayerStats {
	todaysStats := []*models.DailyPlayerStats{}
	searchablePlayers := makePlayersSearchable(players)

	for _, historyRecord := range history {
		if !h.isTheSameAsServerHistoryUpdatedAt(historyRecord.CreateDate) {
			continue
		}
		if index := searchByID(searchablePlayers, historyRecord.PlayerID); index != -1 {
			player := players[index]
			todaysStats = append(todaysStats, &models.DailyPlayerStats{
				PlayerID:          player.ID,
				Villages:          player.TotalVillages - historyRecord.TotalVillages,
				Points:            player.Points - historyRecord.Points,
				Rank:              (player.Rank - historyRecord.Rank) * -1,
				CreateDate:        historyRecord.CreateDate,
				OpponentsDefeated: h.calculateODifference(player.OpponentsDefeated, historyRecord.OpponentsDefeated),
			})
		}
	}

	return todaysStats
}

func (h *updateServerDataWorker) update() error {
	pod, err := h.dataloader.LoadOD(false)
	if err != nil {
		return err
	}
	tod, err := h.dataloader.LoadOD(true)
	if err != nil {
		return err
	}

	ennoblements, err := h.loadEnnoblements()
	if err != nil {
		return err
	}

	villages, err := h.dataloader.LoadVillages()
	if err != nil {
		return err
	}
	numberOfVillages := len(villages)

	tribes, err := h.loadTribes(tod, countPlayerVillages(villages))
	if err != nil {
		return err
	}
	numberOfTribes := len(tribes)

	players, err := h.loadPlayers(pod)
	if err != nil {
		return err
	}
	numberOfPlayers := len(players)

	cfg, err := h.dataloader.GetConfig()
	if err != nil {
		return err
	}
	buildingCfg, err := h.dataloader.GetBuildingConfig()
	if err != nil {
		return err
	}
	unitCfg, err := h.dataloader.GetUnitConfig()
	if err != nil {
		return err
	}

	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	if len(tribes) > 0 {
		ids := []int{}
		for _, tribe := range tribes {
			ids = append(ids, tribe.ID)
		}

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
			Apply(appendODSetClauses).
			Insert(); err != nil {
			return errors.Wrap(err, "couldnt insert tribes")
		}
		if _, err := tx.Model(&tribes).
			Where("NOT (tribe.id  = ANY (?))", pg.Array(ids)).
			Set("exists = false").
			Update(); err != nil && err != pg.ErrNoRows {
			return errors.Wrap(err, "couldnt update nonexistent tribes")
		}

		tribesHistory := []*models.TribeHistory{}
		if err := h.db.Model(&tribesHistory).
			DistinctOn("tribe_id").
			Column("*").
			Where("tribe_id = ANY (?)", pg.Array(ids)).
			Order("tribe_id DESC", "create_date DESC").
			Select(); err != nil && err != pg.ErrNoRows {
			return errors.Wrap(err, "couldnt select tribe history records")
		}
		todaysTribeStats := h.calculateTodaysTribeStats(tribes, tribesHistory)
		if len(todaysTribeStats) > 0 {
			if _, err := tx.
				Model(&todaysTribeStats).
				OnConflict("ON CONSTRAINT daily_tribe_stats_tribe_id_create_date_key DO UPDATE").
				Set("members = EXCLUDED.members").
				Set("villages = EXCLUDED.villages").
				Set("points = EXCLUDED.points").
				Set("all_points = EXCLUDED.all_points").
				Set("rank = EXCLUDED.rank").
				Set("dominance = EXCLUDED.dominance").
				Apply(appendODSetClauses).
				Insert(); err != nil {
				return errors.Wrap(err, "couldnt insert today's tribe stats")
			}
		}
	}
	if len(players) > 0 {
		ids := []int{}
		for _, player := range players {
			ids = append(ids, player.ID)
		}
		if _, err := tx.Model(&players).
			OnConflict("(id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("total_villages = EXCLUDED.total_villages").
			Set("points = EXCLUDED.points").
			Set("rank = EXCLUDED.rank").
			Set("exists = EXCLUDED.exists").
			Set("tribe_id = EXCLUDED.tribe_id").
			Set("daily_growth = EXCLUDED.daily_growth").
			Apply(appendODSetClauses).
			Insert(); err != nil {
			return errors.Wrap(err, "couldnt insert players")
		}
		if _, err := tx.Model(&models.Player{}).
			Where("NOT (player.id  = ANY (?))", pg.Array(ids)).
			Set("exists = false").
			Update(); err != nil && err != pg.ErrNoRows {
			return errors.Wrap(err, "couldnt update nonexistent players")
		}

		playerHistory := []*models.PlayerHistory{}
		if err := h.db.Model(&playerHistory).
			DistinctOn("player_id").
			Column("*").
			Where("player_id = ANY (?)", pg.Array(ids)).
			Order("player_id DESC", "create_date DESC").Select(); err != nil && err != pg.ErrNoRows {
			return errors.Wrap(err, "couldnt select player history records")
		}
		todaysPlayerStats := h.calculateDailyPlayerStats(players, playerHistory)
		if len(todaysPlayerStats) > 0 {
			if _, err := tx.
				Model(&todaysPlayerStats).
				OnConflict("ON CONSTRAINT daily_player_stats_player_id_create_date_key DO UPDATE").
				Set("villages = EXCLUDED.villages").
				Set("points = EXCLUDED.points").
				Set("rank = EXCLUDED.rank").
				Apply(appendODSetClauses).
				Insert(); err != nil {
				return errors.Wrap(err, "couldnt insert today's player stats")
			}
		}

	}
	if len(villages) > 0 {
		if _, err := tx.Model(&models.Village{}).
			Where("true").
			Delete(); err != nil && err != pg.ErrNoRows {
			return errors.Wrap(err, "couldnt delete villages")
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
			return errors.Wrap(err, "couldnt insert villages")
		}
	}
	if len(ennoblements) > 0 {
		if _, err := tx.Model(&ennoblements).Insert(); err != nil {
			return errors.Wrap(err, "couldnt insert ennoblements")
		}
	}

	if _, err := tx.Model(h.server).
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
		return errors.Wrap(err, "couldnt update server")
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func appendODSetClauses(q *orm.Query) (*orm.Query, error) {
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
