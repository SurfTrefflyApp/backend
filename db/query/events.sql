-- name: CreateEvent :one
INSERT INTO events (
    name,
    description,
    capacity,
    latitude,
    longitude,
    address,
    date,
    owner_id,
    is_private,
    image_id
) VALUES (
             @name,
             @description,
             @capacity,
             @latitude,
             @longitude,
             @address,
             @date,
             @owner_id,
             @is_private,
             @image_id
         )
    RETURNING id, name, description, capacity, latitude, longitude,
    address, date, owner_id, is_private, is_premium, created_at, image_id;

-- name: GetEvent :one
SELECT
    e.id,
    e.name,
    e.description,
    e.capacity,
    e.latitude,
    e.longitude,
    e.address,
    e.date,
    e.owner_id,
    e.owner_username,
    e.is_private,
    e.is_premium,
    e.created_at,
    e.tags,
    e.image_id,
    e.participants_count,
    e.event_image_path,
    e.user_image_path,
    CASE
        WHEN $2 = e.owner_id THEN true
        WHEN (SELECT is_admin FROM users WHERE id = $2) THEN true
        WHEN NOT e.is_private THEN true
        WHEN EXISTS (
            SELECT 1 FROM event_user eu
            WHERE eu.event_id = e.id
              AND eu.user_id = $2
        ) THEN true
        WHEN et.token IS NOT NULL THEN
            et.expires_at > NOW()
        ELSE false
        END AS allowed
FROM event_with_tags_view e
         LEFT JOIN event_tokens et
                   ON e.id = et.event_id
                       AND et.token = $3
WHERE e.id = $1
  AND (
    NOT e.is_private
        OR
    (e.is_private AND (
        EXISTS (SELECT 1 FROM event_user WHERE event_id = e.id AND user_id = $2)
            OR et.token IS NOT NULL
            OR $2 = e.owner_id
            OR (SELECT is_admin FROM users WHERE id = $2)
        ))
    );

-- name: ListEvents :many
SELECT
    evt.id,
    evt.name,
    evt.description,
    evt.capacity,
    evt.latitude,
    evt.longitude,
    evt.address,
    evt.date,
    evt.owner_id,
    evt.owner_username,
    evt.is_private,
    evt.is_premium,
    evt.created_at,
    evt.tags,
    evt.participants_count,
    evt.event_image_path,
    evt.user_image_path,
    (
        SELECT COUNT(*)
        FROM event_tags et
        WHERE et.event_id = evt.id
          AND et.tag_id = ANY(@tag_ids::int[])
    ) AS matched_tags,
    ST_Distance(
            evt.geom,
            ST_MakePoint(@user_lon::numeric, @user_lat::numeric)::GEOGRAPHY
    ) AS distance
FROM event_with_tags_view evt
WHERE
    ST_DWithin(
            evt.geom,
            ST_MakePoint(@user_lon::numeric, @user_lat::numeric)::GEOGRAPHY,
            100000
    )
  AND evt.is_private = false
  AND evt.date > NOW()
  AND (
    @search_term::text = ''
        OR (
            evt.name ILIKE '%' || @search_term || '%'
            OR evt.description ILIKE '%' || @search_term || '%'
        )
    )
  AND (
    cardinality(@tag_ids::int[]) = 0
        OR EXISTS (
        SELECT 1
        FROM event_tags et
        WHERE et.event_id = evt.id
          AND et.tag_id = ANY(@tag_ids::int[])
    )
    )
  AND (
    @date_range::text IS NULL
        OR @date_range::text = ''
        OR CASE
            WHEN @date_range = 'day' THEN evt.date BETWEEN NOW() AND NOW() + INTERVAL '1 day'
            WHEN @date_range = 'week' THEN evt.date BETWEEN NOW() AND NOW() + INTERVAL '7 days'
            WHEN @date_range = 'month' THEN evt.date BETWEEN NOW() AND NOW() + INTERVAL '1 month'
            ELSE TRUE
        END
    )
ORDER BY
    CASE WHEN @search_term::text <> '' THEN
             SIMILARITY(evt.name, @search_term) +
             SIMILARITY(evt.description, @search_term)
         ELSE 0 END DESC,
    matched_tags DESC,
    evt.created_at DESC,
    distance ASC;

-- name: UpdateEvent :exec
UPDATE events
SET
    name = @name,
    description = @description,
    capacity = @capacity,
    latitude = @latitude,
    longitude = @longitude,
    address = @address,
    date = @date,
    is_private = @is_private,
    image_id = @image_id
WHERE id = @id;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = $1;

-- name: GetPremiumEvents :many
SELECT
    id,
    name,
    description,
    capacity,
    latitude,
    longitude,
    address,
    date,
    owner_id,
    owner_username,
    is_private,
    is_premium,
    created_at,
    tags,
    participants_count,
    event_image_path,
    user_image_path
FROM event_with_tags_view
WHERE is_premium = TRUE
  AND date > NOW() AND is_private = false
ORDER BY created_at DESC
    LIMIT 9;

-- name: GetLatestEvents :many
SELECT
    id,
    name,
    description,
    capacity,
    latitude,
    longitude,
    address,
    date,
    owner_id,
    owner_username,
    is_private,
    is_premium,
    created_at,
    tags,
    participants_count,
    event_image_path,
    user_image_path
FROM event_with_tags_view
WHERE date > NOW() AND is_private = false
ORDER BY created_at DESC
    LIMIT 9;

-- name: GetPopularEvents :many
SELECT
    id,
    name,
    description,
    capacity,
    latitude,
    longitude,
    address,
    date,
    owner_id,
    owner_username,
    is_private,
    is_premium,
    created_at,
    tags,
    participants_count,
    event_image_path,
    user_image_path
FROM event_with_tags_view
WHERE date > NOW() AND is_private = false
ORDER BY participants_count DESC, created_at DESC
    LIMIT 9;

-- name: GetUserRecommendedEvents :many
WITH user_tags AS (
    SELECT (tag->>'id')::INT AS tag_id
    FROM user_with_tags_view,
         json_array_elements(tags) AS tag
    WHERE user_with_tags_view.id = @user_id
)
SELECT
    evt.id,
    evt.name,
    evt.description,
    evt.capacity,
    evt.latitude,
    evt.longitude,
    evt.address,
    evt.date,
    evt.owner_id,
    evt.owner_username,
    evt.is_private,
    evt.is_premium,
    evt.created_at,
    evt.tags,
    evt.participants_count,
    evt.event_image_path,
    evt.user_image_path,
    (
        SELECT COUNT(*)
        FROM event_tags et
        WHERE
            et.event_id = evt.id
          AND et.tag_id IN (SELECT tag_id FROM user_tags)
    ) AS matched_tags,
    ST_Distance(
            evt.geom,
            ST_MakePoint(@user_lon::numeric, @user_lat::numeric)::GEOGRAPHY
    ) AS distance
FROM event_with_tags_view evt
WHERE
    evt.date > NOW()
  AND ST_DWithin(
        evt.geom,
        ST_MakePoint(@user_lon::numeric, @user_lat::numeric)::GEOGRAPHY,
        100000
      )
  AND evt.is_private = false
ORDER BY
    matched_tags DESC,
    created_at DESC,
    distance ASC
    LIMIT 9;

-- name: GetGuestRecommendedEvents :many
SELECT
    id,
    name,
    description,
    capacity,
    latitude,
    longitude,
    address,
    date,
    owner_id,
    owner_username,
    is_private,
    is_premium,
    created_at,
    tags,
    participants_count,
    event_image_path,
    user_image_path
FROM event_with_tags_view
WHERE
    date > NOW()
  AND ST_DWithin(
    geom,
    ST_MakePoint(@user_lon::numeric, @user_lat::numeric)::GEOGRAPHY,
    100000
    )
  AND is_private = false
ORDER BY
    ST_Distance(geom, ST_MakePoint(@user_lon::numeric, @user_lat::numeric)::GEOGRAPHY) ASC,
    created_at DESC
    LIMIT 9;

-- name: GetPastUserEvents :many
SELECT
    e.id,
    e.name,
    e.description,
    e.capacity,
    e.latitude,
    e.longitude,
    e.address,
    e.date,
    e.owner_id,
    e.owner_username,
    e.is_private,
    e.is_premium,
    e.created_at,
    e.tags,
    e.participants_count,
    e.event_image_path,
    e.user_image_path
FROM event_with_tags_view e
         JOIN event_user eu ON e.id = eu.event_id
WHERE
    eu.user_id = @user_id
  AND e.date < NOW()
ORDER BY
    e.date DESC;

-- name: GetUpcomingUserEvents :many
SELECT
    e.id,
    e.name,
    e.description,
    e.capacity,
    e.latitude,
    e.longitude,
    e.address,
    e.date,
    e.owner_id,
    e.owner_username,
    e.is_private,
    e.is_premium,
    e.created_at,
    e.tags,
    e.participants_count,
    e.event_image_path,
    e.user_image_path
FROM event_with_tags_view e
         JOIN event_user eu ON e.id = eu.event_id
WHERE
    eu.user_id = @user_id
  AND e.date > NOW()
ORDER BY
    e.date ASC;

-- name: GetOwnedUserEvents :many
SELECT
    e.id,
    e.name,
    e.description,
    e.capacity,
    e.latitude,
    e.longitude,
    e.address,
    e.date,
    e.owner_id,
    e.owner_username,
    e.is_private,
    e.is_premium,
    e.created_at,
    e.tags,
    e.participants_count,
    e.event_image_path,
    e.user_image_path
FROM event_with_tags_view e
WHERE
    e.owner_id = @user_id
ORDER BY
    e.date DESC;

-- name: CreatePrivateEventToken :exec
INSERT INTO event_tokens (
    event_id,
    token,
    expires_at
) VALUES ($1, $2, $3);

-- name: ListAllEvents :many
SELECT
    evt.id,
    evt.name,
    evt.description,
    evt.capacity,
    evt.latitude,
    evt.longitude,
    evt.address,
    evt.date,
    evt.owner_id,
    evt.owner_username,
    evt.is_private,
    evt.is_premium,
    evt.created_at,
    evt.tags,
    evt.participants_count,
    evt.event_image_path,
    evt.user_image_path,
    (
        SELECT COUNT(*)
        FROM event_tags et
        WHERE et.event_id = evt.id
          AND et.tag_id = ANY(@tag_ids::int[])
    ) AS matched_tags,
    NULL::float AS distance
FROM event_with_tags_view evt
WHERE
    evt.date > NOW()
  AND (
    @search_term::text = ''
        OR (
            evt.name ILIKE '%' || @search_term || '%'
            OR evt.description ILIKE '%' || @search_term || '%'
        )
    )
  AND (
    cardinality(@tag_ids::int[]) = 0
        OR EXISTS (
        SELECT 1
        FROM event_tags et
        WHERE et.event_id = evt.id
          AND et.tag_id = ANY(@tag_ids::int[])
    )
    )
  AND (
    @date_range::text IS NULL
        OR @date_range::text = ''
        OR CASE
            WHEN @date_range = 'day' THEN evt.date BETWEEN NOW() AND NOW() + INTERVAL '1 day'
            WHEN @date_range = 'week' THEN evt.date BETWEEN NOW() AND NOW() + INTERVAL '7 days'
            WHEN @date_range = 'month' THEN evt.date BETWEEN NOW() AND NOW() + INTERVAL '1 month'
            ELSE TRUE
        END
    )
ORDER BY
    CASE WHEN @search_term::text <> '' THEN
             SIMILARITY(evt.name, @search_term) +
             SIMILARITY(evt.description, @search_term)
         ELSE 0 END DESC,
    matched_tags DESC,
    evt.created_at DESC;

-- name: CreatePremiumOrder :one
INSERT INTO premium_orders (
    user_id,
    event_id,
    shop,
    price
) VALUES (
    $1, $2, $3, $4
         ) RETURNING *;

-- name: GetPremiumOrder :one
SELECT * FROM premium_orders
WHERE id = $1;

-- name: SetEventPremium :exec
WITH
    get_event AS (
        SELECT event_id
        FROM premium_orders
        WHERE premium_orders.id = $1
    ),
    update_event AS (
UPDATE events
SET is_premium = true
WHERE id = (SELECT event_id FROM get_event)
    ),
update_order AS (
    UPDATE premium_orders
    SET status = 'complete'
    WHERE id = $1
)
SELECT 1;