-- 用户中心收藏别人帖子列表测试数据
-- 用法：
-- 1. 把下面的 v_user_id 改成你要测试的真实用户 ID
-- 2. 在 PostgreSQL 中执行本脚本
-- 3. 执行后可调用：
--    GET /api/user/center/favorites?page=1&limit=10
--
-- 本脚本会造两类数据：
-- 1. 命中接口的数据：当前用户收藏了“别人发的帖子”
-- 2. 不命中的对照数据：收藏了自己的帖子 / 非 topic 收藏 / 帖子状态非正常

BEGIN;

DO $$
DECLARE
    v_user_id bigint := 1;
    v_other_user_id bigint := 1004;
    v_now bigint := EXTRACT(EPOCH FROM NOW())::bigint;
    v_topic_other_01_id bigint;
    v_topic_other_02_id bigint;
    v_topic_self_id bigint;
    v_topic_deleted_id bigint;
BEGIN
    INSERT INTO public.t_topic (
        type,
        node_id,
        user_id,
        title,
        content_type,
        content,
        image_list,
        hide_content,
        vote_id,
        recommend,
        recommend_time,
        sticky,
        sticky_time,
        view_count,
        comment_count,
        like_count,
        status,
        last_comment_time,
        last_comment_user_id,
        user_agent,
        ip,
        ip_location,
        create_time,
        extra_data
    ) VALUES (
        0,
        1,
        v_other_user_id,
        '用户中心收藏测试别人帖子 01',
        'markdown',
        '这是给用户中心收藏列表接口准备的别人帖子 01。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        15,
        3,
        2,
        0,
        v_now - 600,
        v_other_user_id,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 600,
        ''
    )
    RETURNING id INTO v_topic_other_01_id;

    INSERT INTO public.t_topic (
        type,
        node_id,
        user_id,
        title,
        content_type,
        content,
        image_list,
        hide_content,
        vote_id,
        recommend,
        recommend_time,
        sticky,
        sticky_time,
        view_count,
        comment_count,
        like_count,
        status,
        last_comment_time,
        last_comment_user_id,
        user_agent,
        ip,
        ip_location,
        create_time,
        extra_data
    ) VALUES (
        0,
        1,
        v_other_user_id,
        '用户中心收藏测试别人帖子 02',
        'markdown',
        '这是给用户中心收藏列表接口准备的别人帖子 02。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        8,
        1,
        5,
        0,
        v_now - 500,
        v_other_user_id,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 500,
        ''
    )
    RETURNING id INTO v_topic_other_02_id;

    INSERT INTO public.t_topic (
        type,
        node_id,
        user_id,
        title,
        content_type,
        content,
        image_list,
        hide_content,
        vote_id,
        recommend,
        recommend_time,
        sticky,
        sticky_time,
        view_count,
        comment_count,
        like_count,
        status,
        last_comment_time,
        last_comment_user_id,
        user_agent,
        ip,
        ip_location,
        create_time,
        extra_data
    ) VALUES (
        0,
        1,
        v_user_id,
        '用户中心收藏测试自己的帖子',
        'markdown',
        '这是当前用户自己发的帖子，用于验证收藏自己帖子不会命中接口。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        6,
        0,
        1,
        0,
        v_now - 400,
        v_user_id,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 400,
        ''
    )
    RETURNING id INTO v_topic_self_id;

    INSERT INTO public.t_topic (
        type,
        node_id,
        user_id,
        title,
        content_type,
        content,
        image_list,
        hide_content,
        vote_id,
        recommend,
        recommend_time,
        sticky,
        sticky_time,
        view_count,
        comment_count,
        like_count,
        status,
        last_comment_time,
        last_comment_user_id,
        user_agent,
        ip,
        ip_location,
        create_time,
        extra_data
    ) VALUES (
        0,
        1,
        v_other_user_id,
        '用户中心收藏测试已删除帖子',
        'markdown',
        '这是状态非正常的帖子，用于验证不会出现在收藏列表中。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        2,
        0,
        0,
        1,
        v_now - 300,
        v_other_user_id,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 300,
        ''
    )
    RETURNING id INTO v_topic_deleted_id;

    -- 命中数据：当前用户收藏别人发的帖子
    INSERT INTO public.t_favorite (
        user_id,
        entity_type,
        entity_id,
        create_time
    ) VALUES
    (
        v_user_id,
        'topic',
        v_topic_other_01_id,
        v_now - 120
    ),
    (
        v_user_id,
        'topic',
        v_topic_other_02_id,
        v_now - 60
    );

    -- 不命中对照数据 1：收藏自己的帖子
    INSERT INTO public.t_favorite (
        user_id,
        entity_type,
        entity_id,
        create_time
    ) VALUES (
        v_user_id,
        'topic',
        v_topic_self_id,
        v_now - 40
    );

    -- 不命中对照数据 2：非 topic 类型收藏
    INSERT INTO public.t_favorite (
        user_id,
        entity_type,
        entity_id,
        create_time
    ) VALUES (
        v_user_id,
        'article',
        999999,
        v_now - 20
    );

    -- 不命中对照数据 3：收藏状态非正常的帖子
    INSERT INTO public.t_favorite (
        user_id,
        entity_type,
        entity_id,
        create_time
    ) VALUES (
        v_user_id,
        'topic',
        v_topic_deleted_id,
        v_now - 10
    );
END $$;

COMMIT;
