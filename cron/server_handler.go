package cron

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/tribalwarshelp/shared/models"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
)

const (
	endpointPlayers      = "/map/player.txt"
	endpointTribe        = "/map/ally.txt"
	endpointVillage      = "/map/village.txt"
	endpointKillAtt      = "/map/kill_att.txt"
	endpointKillDef      = "/map/kill_def.txt"
	endpointKillSup      = "/map/kill_sup.txt"
	endpointKillAll      = "/map/kill_all.txt"
	endpointKillAttTribe = "/map/kill_att_tribe.txt"
	endpointKillDefTribe = "/map/kill_def_tribe.txt"
	endpointKillAllTribe = "/map/kill_all_tribe.txt"
)

type serverHandler struct {
	baseURL string
	db      *pg.DB
}

type parsedODLine struct {
	ID    int
	Rank  int
	Score int
}

func (h *serverHandler) parseODLine(line []string) (*parsedODLine, error) {
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

func (h *serverHandler) getOD(tribe bool) (map[int]*models.OpponentsDefeated, error) {
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
		}
	}
	for _, url := range urls {
		lines, err := getCSVData(url, false)
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
			if url == urls[0] {
				m[parsed.ID].RankTotal = parsed.Rank
				m[parsed.ID].ScoreTotal = parsed.Score
			} else if url == urls[1] {
				m[parsed.ID].RankAtt = parsed.Rank
				m[parsed.ID].ScoreAtt = parsed.Score
			} else if url == urls[2] {
				m[parsed.ID].RankDef = parsed.Rank
				m[parsed.ID].ScoreDef = parsed.Score
			} else if !tribe && url == urls[3] {
				m[parsed.ID].RankSup = parsed.Rank
				m[parsed.ID].ScoreSup = parsed.Score
			}
		}
	}
	return m, nil
}

func (h *serverHandler) parsePlayerLine(line []string) (*models.Player, error) {
	if len(line) != 6 {
		return nil, fmt.Errorf("Invalid line format (should be id,name,tribeid,villages,points,rank)")
	}

	var err error
	ex := true
	player := &models.Player{
		Exist: &ex,
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

func (h *serverHandler) getPlayers(od map[int]*models.OpponentsDefeated) ([]*models.Player, error) {
	url := h.baseURL + endpointPlayers
	lines, err := getCSVData(url, false)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get data, url %s", url)
	}

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
		players = append(players, player)
	}

	return players, nil
}

func (h *serverHandler) parseTribeLine(line []string) (*models.Tribe, error) {
	if len(line) != 8 {
		return nil, fmt.Errorf("Invalid line format (should be id,name,tag,members,villages,points,allpoints,rank)")
	}

	var err error
	ex := true
	tribe := &models.Tribe{
		Exist: &ex,
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

func (h *serverHandler) getTribes(od map[int]*models.OpponentsDefeated) ([]*models.Tribe, error) {
	url := h.baseURL + endpointTribe
	lines, err := getCSVData(url, false)
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
		tribes = append(tribes, tribe)
	}
	return tribes, nil
}

func (h *serverHandler) parseVillageLine(line []string) (*models.Village, error) {
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

func (h *serverHandler) getVillages() ([]*models.Village, error) {
	url := h.baseURL + endpointVillage
	lines, err := getCSVData(url, false)
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

func (h *serverHandler) updateData() error {
	pod, err := h.getOD(false)
	if err != nil {
		return err
	}
	tod, err := h.getOD(true)
	if err != nil {
		return err
	}
	tribes, err := h.getTribes(tod)
	if err != nil {
		return err
	}
	players, err := h.getPlayers(pod)
	if err != nil {
		return err
	}
	villages, err := h.getVillages()
	if err != nil {
		return err
	}

	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	if len(tribes) > 0 {
		if _, err := tx.Model(&tribes).
			OnConflict("(id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("tag = EXCLUDED.tag").
			Set("total_members = EXCLUDED.total_members").
			Set("total_villages = EXCLUDED.total_villages").
			Set("points = EXCLUDED.points").
			Set("rank = EXCLUDED.rank").
			Set("exist = EXCLUDED.exist").
			Apply(attachODSetClauses).
			Insert(); err != nil {
			return errors.Wrap(err, "cannot insert tribes")
		}

		ids := []int{}
		for _, tribe := range tribes {
			ids = append(ids, tribe.ID)
		}
		if _, err := tx.Model(&models.Tribe{}).
			Where("id NOT IN (?)", pg.In(ids)).
			Set("exist = false").
			Update(); err != nil && err != pg.ErrNoRows {
			return errors.Wrap(err, "cannot update not existed tribes")
		}
	}
	if len(players) > 0 {
		if _, err := tx.Model(&players).
			OnConflict("(id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("total_villages = EXCLUDED.total_villages").
			Set("points = EXCLUDED.points").
			Set("rank = EXCLUDED.rank").
			Set("exist = EXCLUDED.exist").
			Set("tribe_id = EXCLUDED.tribe_id").
			Apply(attachODSetClauses).
			Insert(); err != nil {
			return errors.Wrap(err, "cannot insert players")
		}

		ids := []int{}
		for _, player := range players {
			ids = append(ids, player.ID)
		}
		if _, err := tx.Model(&models.Player{}).
			Where("id NOT IN (?)", pg.In(ids)).
			Set("exist = false").
			Update(); err != nil && err != pg.ErrNoRows {
			return errors.Wrap(err, "cannot update not existed players")
		}
	}
	if len(villages) > 0 {
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

		ids := []int{}
		for _, village := range villages {
			ids = append(ids, village.ID)
		}
		if _, err := tx.Model(&models.Village{}).
			Where("id NOT IN (?)", pg.In(ids)).
			Delete(); err != nil && err != pg.ErrNoRows {
			return errors.Wrap(err, "cannot delete not existed villages")
		}
	}

	return tx.Commit()
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
