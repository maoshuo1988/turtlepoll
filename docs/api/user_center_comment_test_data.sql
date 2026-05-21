-- 用户中心评论列表测试数据
-- 用法：
-- 1. 把下面的 v_user_id 改成你要测试的真实用户 ID
-- 2. 在 PostgreSQL 中执行本脚本
-- 3. 执行后可调用：
--    GET /api/user/center/comments?user=<v_user_id>&page=1&limit=10
--
-- 本脚本会造两类数据：
-- 1. 命中接口的数据：当前用户评论了“别人发的帖子”
-- 2. 不命中的对照数据：评论了自己的帖子 / 非 topic 评论 / 非正常状态评论

BEGIN;

DO $$
DECLARE
    v_user_id bigint := 1;
    v_other_user_id bigint := 1004;
    v_now bigint := EXTRACT(EPOCH FROM NOW())::bigint;
    v_topic_other_01_id bigint;
    v_topic_other_02_id bigint;
    v_topic_self_id bigint;
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
        '用户中心评论测试别人帖子 01',
        'markdown',
        '这是给用户中心评论列表接口准备的别人帖子 01。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 500,
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
        '用户中心评论测试别人帖子 02',
        'markdown',
        '这是给用户中心评论列表接口准备的别人帖子 02。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 400,
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
        '用户中心评论测试自己的帖子',
        'markdown',
        '这是给用户中心评论列表接口准备的本人帖子，不应命中评论列表。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 300,
        ''
    )
    RETURNING id INTO v_topic_self_id;

    -- 命中数据：当前用户评论别人发的帖子
    INSERT INTO public.t_comment (
        user_id,
        entity_type,
        entity_id,
        content,
        image_list,
        content_type,
        quote_id,
        like_count,
        comment_count,
        user_agent,
        ip,
        ip_location,
        status,
        create_time
    ) VALUES
    (
        v_user_id,
        'topic',
        v_topic_other_01_id,
        '这是用户中心评论列表测试评论 01，会命中接口结果。',
        '',
        'text',
        0,
        0,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        0,
        v_now - 120
    ),
    (
        v_user_id,
        'topic',
        v_topic_other_02_id,
        '这是用户中心评论列表测试评论 02，会命中接口结果，并且因为时间更新会排在更前面。',
        '',
        'text',
        0,
        0,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        0,
        v_now - 60
    );

    -- 不命中对照数据 1：评论自己的帖子
    INSERT INTO public.t_comment (
        user_id,
        entity_type,
        entity_id,
        content,
        image_list,
        content_type,
        quote_id,
        like_count,
        comment_count,
        user_agent,
        ip,
        ip_location,
        status,
        create_time
    ) VALUES (
        v_user_id,
        'topic',
        v_topic_self_id,
        '这是评论自己帖子的测试数据，不应该出现在接口结果里。',
        '',
        'text',
        0,
        0,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        0,
        v_now - 30
    );

    -- 不命中对照数据 2：entity_type 不是 topic
    INSERT INTO public.t_comment (
        user_id,
        entity_type,
        entity_id,
        content,
        image_list,
        content_type,
        quote_id,
        like_count,
        comment_count,
        user_agent,
        ip,
        ip_location,
        status,
        create_time
    ) VALUES (
        v_user_id,
        'comment',
        999999,
        '这是非 topic 类型的评论测试数据，不应该出现在接口结果里。',
        '',
        'text',
        0,
        0,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        0,
        v_now - 20
    );

    -- 不命中对照数据 3：评论状态不是正常
    INSERT INTO public.t_comment (
        user_id,
        entity_type,
        entity_id,
        content,
        image_list,
        content_type,
        quote_id,
        like_count,
        comment_count,
        user_agent,
        ip,
        ip_location,
        status,
        create_time
    ) VALUES (
        v_user_id,
        'topic',
        v_topic_other_01_id,
        '这是非正常状态的评论测试数据，不应该出现在接口结果里。',
        '',
        'text',
        0,
        0,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        1,
        v_now - 10
    );

    UPDATE public.t_topic
    SET
        comment_count = 2,
        last_comment_time = v_now - 120,
        last_comment_user_id = v_user_id
    WHERE id = v_topic_other_01_id;

    UPDATE public.t_topic
    SET
        comment_count = 1,
        last_comment_time = v_now - 60,
        last_comment_user_id = v_user_id
    WHERE id = v_topic_other_02_id;

    UPDATE public.t_topic
    SET
        comment_count = 1,
        last_comment_time = v_now - 30,
        last_comment_user_id = v_user_id
    WHERE id = v_topic_self_id;
END $$;

COMMIT;
