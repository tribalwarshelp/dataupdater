package db

const (
	allSpecialServersPGInsertStatements = `
		INSERT INTO public.special_servers (version_code, key) VALUES ('pl', 'pls1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('en', 'ens1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('uk', 'uks1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('uk', 'master') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('it', 'its1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('hu', 'hus1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('fr', 'frs1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('us', 'uss1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('nl', 'nls1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('es', 'ess1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('ro', 'ros1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('gr', 'grs1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('br', 'brs1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('tr', 'trs1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('cs', 'css1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('de', 'des1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('ru', 'rus1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('ch', 'chs1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('pt', 'pts1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
		INSERT INTO public.special_servers (version_code, key) VALUES ('sk', 'sks1') ON CONFLICT ON CONSTRAINT special_servers_version_code_key_key DO NOTHING;
	`
	allVersionsPGInsertStatements = `
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('pl', 'Polska', 'plemiona.pl', 'Europe/Warsaw') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('uk', 'United Kingdom', 'tribalwars.co.uk', 'Europe/London') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('hu', 'Hungary', 'klanhaboru.hu', 'Europe/Budapest') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('it', 'Italy', 'tribals.it', 'Europe/Rome') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('fr', 'France', 'guerretribale.fr', 'Europe/Paris') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('us', 'United States', 'tribalwars.us', 'America/New_York') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('nl', 'The Netherlands', 'tribalwars.nl', 'Europe/Amsterdam') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('es', 'Spain', 'guerrastribales.es', 'Europe/Madrid') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('ro', 'Romania', 'triburile.ro', 'Europe/Bucharest') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('gr', 'Greece', 'fyletikesmaxes.gr', 'Europe/Athens') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('br', 'Brazil', 'tribalwars.com.br', 'America/Sao_Paulo') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('tr', 'Turkey', 'klanlar.org', 'Europe/Istanbul') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('cs', 'Czech Republic', 'divokekmeny.cz', 'Europe/Prague') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('ru', 'Russia', 'voyna-plemyon.ru', 'Europe/Moscow') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('ch', 'Switerzland', 'staemme.ch', 'Europe/Zurich') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('pt', 'Portugal', 'tribalwars.com.pt', 'Europe/Lisbon') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('en', 'International', 'tribalwars.net', 'Europe/London') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('de', 'Germany', 'die-staemme.de', 'Europe/Berlin') ON CONFLICT (code) DO NOTHING;
		INSERT INTO public.versions (code, name, host, timezone) VALUES ('sk', 'Slovakia', 'divoke-kmene.sk', 'Europe/Bratislava') ON CONFLICT (code) DO NOTHING;
	`

	pgDropSchemaFunctions = `
		DO
		$do$
		DECLARE
				funcTableRecord RECORD;
		BEGIN
			FOR funcTableRecord IN SELECT routine_schema, routine_name from information_schema.routines where routine_type = 'FUNCTION' AND specific_schema = '?0' LOOP
				EXECUTE 'DROP FUNCTION IF EXISTS ' || funcTableRecord.routine_schema || '.' || funcTableRecord.routine_name || ' CASCADE;';
			END LOOP;
		END
		$do$;
	`

	pgFunctions = `
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

		CREATE OR REPLACE FUNCTION update_most_points_most_villages_best_rank_last_activity()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF TG_OP = 'INSERT' THEN
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
			END IF;
			
			IF TG_OP = 'UPDATE' THEN
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
				if TG_TABLE_NAME = 'players' THEN
					IF NEW.points > OLD.points OR NEW.score_att > OLD.score_att THEN
						NEW.last_activity_at = now();
					END IF;
				END IF;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;
	`

	serverPGFunctions = `
		CREATE OR REPLACE FUNCTION ?0.log_tribe_change()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF TG_OP = 'INSERT' THEN
				IF NEW.tribe_id <> 0 THEN
					INSERT INTO ?0.tribe_changes(player_id,old_tribe_id,new_tribe_id,created_at)
					VALUES(NEW.id,0,NEW.tribe_id,now());
				END IF;
			END IF;

			IF TG_OP = 'UPDATE' THEN
				IF NEW.tribe_id <> OLD.tribe_id THEN
					INSERT INTO ?0.tribe_changes(player_id,old_tribe_id,new_tribe_id,created_at)
					VALUES(OLD.id,OLD.tribe_id,NEW.tribe_id,now());
				END IF;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql VOLATILE;

		CREATE OR REPLACE FUNCTION ?0.log_player_name_change()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.name <> OLD.name AND old.exists = true THEN
				INSERT INTO player_name_changes(version_code,player_id,old_name,new_name,change_date)
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
	`
	serverPGTriggers = `
		CREATE TRIGGER ?0_log_tribe_change_on_insert
			AFTER INSERT
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.log_tribe_change();
	
		CREATE TRIGGER ?0_log_tribe_change_on_update
			AFTER UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.log_tribe_change();

		CREATE TRIGGER ?0_name_change
			AFTER UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.log_player_name_change();

		CREATE TRIGGER ?0_check_daily_growth
			BEFORE UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE check_daily_growth();

		CREATE TRIGGER ?0_check_player_existence
			BEFORE UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE check_existence();

		CREATE TRIGGER ?0_check_tribe_existence
			BEFORE UPDATE
			ON ?0.tribes
			FOR EACH ROW
			EXECUTE PROCEDURE check_existence();

		CREATE TRIGGER ?0_check_dominance
			BEFORE UPDATE
			ON ?0.tribes
			FOR EACH ROW
			EXECUTE PROCEDURE check_dominance();

		CREATE TRIGGER ?0_update_ennoblement_old_and_new_owner_tribe_id
			BEFORE INSERT
			ON ?0.ennoblements
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.get_old_and_new_owner_tribe_id();

		DROP TRIGGER IF EXISTS ?0_insert_into_player_to_servers ON ?0.players;

		CREATE TRIGGER ?0_update_most_points_most_villages_best_rank_last_activity
			BEFORE INSERT OR UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE update_most_points_most_villages_best_rank_last_activity();

		CREATE TRIGGER ?0_update_most_points_most_villages_best_rank_last_activity
			BEFORE INSERT OR UPDATE
			ON ?0.tribes
			FOR EACH ROW
			EXECUTE PROCEDURE update_most_points_most_villages_best_rank_last_activity();
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
