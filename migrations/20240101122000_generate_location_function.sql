-- migrate:up
CREATE OR REPLACE function generate_location ()
RETURNS trigger LANGUAGE plpgsql as $$
BEGIN
    new.location := ST_SetSRID(ST_MakePoint(new.lat,new.lng), 4326);
    return new;
END $$;

CREATE TRIGGER generate_location_trigger
BEFORE INSERT OR UPDATE ON latest_locations
FOR each ROW EXECUTE PROCEDURE generate_location();

-- migrate:down
DROP TRIGGER generate_location_trigger ON latest_locations;

DROP FUNCTION generate_location;
