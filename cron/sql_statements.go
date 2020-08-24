package cron

const (
	allSpecialServersPGInsertStatements = `
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('pl', 'pls1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('en', 'ens1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('uk', 'uks1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('it', 'its1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('hu', 'hus1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('fr', 'frs1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('us', 'uss1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('nl', 'nls1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('es', 'ess1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('ro', 'ros1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('gr', 'grs1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('br', 'brs1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('tr', 'trs1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('cs', 'css1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('de', 'des1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('ru', 'rus1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
		INSERT INTO public.special_servers (lang_version_tag, key) VALUES ('ch', 'chs1') ON CONFLICT ON CONSTRAINT special_servers_lang_version_tag_key_key DO NOTHING;
	`
	allVersionsPGInsertStatements = `
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('pl', 'Polska', 'plemiona.pl', 'Europe/Warsaw') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('de', 'Deutschland', 'die-staemme.de', 'Europe/Berlin') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('uk', 'United Kingdom', 'tribalwars.co.uk', 'Europe/London') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('hu', 'Hungary', 'klanhaboru.hu', 'Europe/Budapest') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('it', 'Italy', 'tribals.it', 'Europe/Rome') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('fr', 'France', 'guerretribale.fr', 'Europe/Paris') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('us', 'United States', 'tribalwars.us', 'America/Washington') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('nl', 'The Netherlands', 'tribalwars.nl', 'Europe/Amsterdam') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('es', 'Spain', 'guerrastribales.es', 'Europe/Madrid') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('ro', 'Romania', 'triburile.ro', 'Europe/Bucharest') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('gr', 'Greece', 'fyletikesmaxes.gr', 'Europe/Athens') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('br', 'Brazil', 'tribalwars.com.br', 'America/Sao_Paulo') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('tr', 'Turkey', 'klanlar.org', 'Europe/Instanbul') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('cs', 'Czech Republic', 'divokekmeny.cz', 'Europe/Prague') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('en', 'Germany', 'tribalwars.net', 'Europe/London') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('ru', 'Russia', 'voyna-plemyon.ru', 'Europe/Moscow') ON CONFLICT (tag) DO NOTHING;
		INSERT INTO public.lang_versions (tag, name, host, timezone) VALUES ('ch', 'Switerzland', 'staemme.ch', 'Europe/Zurich') ON CONFLICT (tag) DO NOTHING;
	`
	serverPGFunctions = `
		CREATE OR REPLACE FUNCTION ?0.log_tribe_change_on_insert()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.tribe_id <> 0 THEN
				INSERT INTO ?0.tribe_changes(player_id,old_tribe_id,new_tribe_id,created_at)
				VALUES(OLD.id,0,NEW.tribe_id,now());
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql VOLATILE;

		CREATE OR REPLACE FUNCTION ?0.log_tribe_change_on_update()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.tribe_id <> OLD.tribe_id THEN
				INSERT INTO ?0.tribe_changes(player_id,old_tribe_id,new_tribe_id,created_at)
				VALUES(OLD.id,OLD.tribe_id,NEW.tribe_id,now());
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql VOLATILE;

		CREATE OR REPLACE FUNCTION ?0.log_player_name_change()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.name <> OLD.name THEN
				INSERT INTO player_name_changes(lang_version_tag,player_id,old_name,new_name,change_date)
				VALUES(?1,NEW.id,OLD.name,NEW.name,CURRENT_DATE)
				ON CONFLICT DO NOTHING;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql VOLATILE;

		CREATE OR REPLACE FUNCTION ?0.get_old_and_new_owner_tribe_id()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.old_owner_id <> 0 THEN
				SELECT tribe_id INTO NEW.old_owner_tribe_id
					FROM ?0.players
					WHERE id = NEW.old_owner_id;
			END IF;
			IF NEW.old_owner_tribe_id IS NULL THEN
				NEW.old_owner_tribe_id = 0;
			END IF;
			IF NEW.new_owner_id <> 0 THEN
				SELECT tribe_id INTO NEW.new_owner_tribe_id
					FROM ?0.players
					WHERE id = NEW.new_owner_id;
			END IF;
			IF NEW.new_owner_tribe_id IS NULL THEN
				NEW.new_owner_tribe_id = 0;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql VOLATILE;

		CREATE OR REPLACE FUNCTION check_daily_growth()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.exists = false THEN
				NEW.daily_growth = 0;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;

		CREATE OR REPLACE FUNCTION check_existence()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.exists = false AND OLD.exists = true THEN
				NEW.deleted_at = now();
			END IF;
			IF NEW.exists = true THEN
				NEW.deleted_at = null;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;

		CREATE OR REPLACE FUNCTION check_dominance()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.exists = false THEN
				NEW.dominance = 0;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;

		CREATE OR REPLACE FUNCTION check_most_points_most_villages_best_rank_values()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.most_points IS null OR NEW.points > NEW.most_points THEN
				NEW.most_points = NEW.points;
				NEW.most_points_at = now();
			END IF;
			IF NEW.most_villages IS null OR NEW.total_villages > NEW.most_villages THEN
				NEW.most_villages = NEW.total_villages;
				NEW.most_villages_at = now();
			END IF;
			IF NEW.best_rank IS null OR NEW.rank < NEW.best_rank OR NEW.best_rank = 0 THEN
				NEW.best_rank = NEW.rank;
				NEW.best_rank_at = now();
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;

		CREATE OR REPLACE FUNCTION update_most_points_most_villages_best_rank()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.most_points IS null OR NEW.points > OLD.most_points THEN
				NEW.most_points = NEW.points;
				NEW.most_points_at = now();
			END IF;
			IF NEW.most_villages IS null OR NEW.total_villages > OLD.most_villages THEN
				NEW.most_villages = NEW.total_villages;
				NEW.most_villages_at = now();
			END IF;
			IF NEW.best_rank IS null OR NEW.rank < OLD.best_rank OR OLD.best_rank = 0 THEN
				NEW.best_rank = NEW.rank;
				NEW.best_rank_at = now();
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;

		CREATE OR REPLACE FUNCTION ?0.insert_to_player_to_servers()
			RETURNS trigger AS
		$BODY$
		BEGIN
			INSERT INTO player_to_servers(server_key,player_id)
				VALUES('?0', NEW.id)
				ON CONFLICT DO NOTHING;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;
	`
	serverPGTriggers = `
		DROP TRIGGER IF EXISTS ?0_log_tribe_change_on_insert ON ?0.players;
		CREATE TRIGGER ?0_log_tribe_change_on_insert
			AFTER INSERT
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.log_tribe_change_on_insert();
	
		DROP TRIGGER IF EXISTS ?0_tribe_changes ON ?0.players;
		DROP TRIGGER IF EXISTS ?0_log_tribe_change_on_update ON ?0.players;
		CREATE TRIGGER ?0_log_tribe_change_on_update
			AFTER UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.log_tribe_change_on_update();

		DROP TRIGGER IF EXISTS ?0_name_change ON ?0.players;
		CREATE TRIGGER ?0_name_change
			AFTER UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.log_player_name_change();

		DROP TRIGGER IF EXISTS ?0_check_daily_growth ON ?0.players;
		CREATE TRIGGER ?0_check_daily_growth
			BEFORE UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE check_daily_growth();

		DROP TRIGGER IF EXISTS ?0_check_player_existence ON ?0.players;
		CREATE TRIGGER ?0_check_player_existence
			BEFORE UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE check_existence();

		DROP TRIGGER IF EXISTS ?0_check_tribe_existence ON ?0.tribes;
		CREATE TRIGGER ?0_check_tribe_existence
			BEFORE UPDATE
			ON ?0.tribes
			FOR EACH ROW
			EXECUTE PROCEDURE check_existence();

		DROP TRIGGER IF EXISTS ?0_check_dominance ON ?0.tribes;
		CREATE TRIGGER ?0_check_dominance
			BEFORE UPDATE
			ON ?0.tribes
			FOR EACH ROW
			EXECUTE PROCEDURE check_dominance();

		DROP TRIGGER IF EXISTS ?0_update_ennoblement_old_and_new_owner_tribe_id ON ?0.ennoblements;
		CREATE TRIGGER ?0_update_ennoblement_old_and_new_owner_tribe_id
			BEFORE INSERT
			ON ?0.ennoblements
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.get_old_and_new_owner_tribe_id();

		DROP TRIGGER IF EXISTS ?0_insert_to_player_to_servers ON ?0.players;
		CREATE TRIGGER ?0_insert_to_player_to_servers
			AFTER INSERT
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.insert_to_player_to_servers();

		DROP TRIGGER IF EXISTS ?0_update_most_points_most_villages_best_rank ON ?0.players;
		CREATE TRIGGER ?0_update_most_points_most_villages_best_rank
			BEFORE UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE update_most_points_most_villages_best_rank();

		DROP TRIGGER IF EXISTS ?0_check_points_villages_rank ON ?0.players;
		CREATE TRIGGER ?0_check_points_villages_rank
			BEFORE INSERT
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE check_most_points_most_villages_best_rank_values();

		DROP TRIGGER IF EXISTS ?0_update_most_points_most_villages_best_rank ON ?0.tribes;
		CREATE TRIGGER ?0_update_most_points_most_villages_best_rank
			BEFORE UPDATE
			ON ?0.tribes
			FOR EACH ROW
			EXECUTE PROCEDURE update_most_points_most_villages_best_rank();

		DROP TRIGGER IF EXISTS ?0_check_points_villages_rank ON ?0.tribes;
		CREATE TRIGGER ?0_check_points_villages_rank
			BEFORE INSERT
			ON ?0.tribes
			FOR EACH ROW
			EXECUTE PROCEDURE check_most_points_most_villages_best_rank_values();
	`

	serverPGDefaultValues = `
		ALTER TABLE ?0.daily_player_stats ALTER COLUMN create_date set default CURRENT_DATE;
		ALTER TABLE ?0.daily_tribe_stats ALTER COLUMN create_date set default CURRENT_DATE;
		ALTER TABLE ?0.player_history ALTER COLUMN create_date set default CURRENT_DATE;
		ALTER TABLE ?0.tribe_history ALTER COLUMN create_date set default CURRENT_DATE;
		ALTER TABLE ?0.stats ALTER COLUMN create_date set default CURRENT_DATE;
	`

	pgDefaultValues = `
		ALTER TABLE player_name_changes ALTER COLUMN change_date set default CURRENT_DATE;
	`
)
