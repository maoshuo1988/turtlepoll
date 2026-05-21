-- 标签评论统计接口测试数据
-- 统计链路：t_tag -> t_topic_tag -> t_topic -> t_comment
-- 统计条件：t_comment.entity_type = 'topic' AND t_comment.entity_id = t_topic.id

BEGIN;

-- 1. 标签
INSERT INTO t_tag (id, name, description, status, create_time, update_time) VALUES
  (900001, '英超', '英超讨论', 0, 1710000000, 1710000000),
  (900002, '西甲', '西甲讨论', 0, 1710000001, 1710000001),
  (900003, '欧冠', '欧冠讨论', 0, 1710000002, 1710000002),
  (900004, '德甲', '德甲讨论', 0, 1710000003, 1710000003)
ON CONFLICT (id) DO NOTHING;

-- 2. 话题
INSERT INTO t_topic (
  id, type, node_id, user_id, title, content_type, content, image_list, hide_content, vote_id,
  recommend, recommend_time, sticky, sticky_time, view_count, comment_count, like_count, status,
  last_comment_time, last_comment_user_id, user_agent, ip, ip_location, create_time, extra_data
) VALUES
  (910001, 0, 1, 1001, '英超第1轮讨论', 'markdown', '讨论英超第1轮', NULL, NULL, 0, false, 0, false, 0, 10, 4, 1, 0, 1710001000, 2004, 'seed', '127.0.0.1', 'Local', 1710000100, NULL),
  (910002, 0, 1, 1002, '西甲争冠分析', 'markdown', '讨论西甲争冠', NULL, NULL, 0, false, 0, false, 0, 20, 2, 3, 0, 1710001100, 2006, 'seed', '127.0.0.1', 'Local', 1710000200, NULL),
  (910003, 0, 1, 1003, '欧冠半决赛前瞻', 'markdown', '讨论欧冠半决赛', NULL, NULL, 0, false, 0, false, 0, 30, 5, 5, 0, 1710001200, 2009, 'seed', '127.0.0.1', 'Local', 1710000300, NULL),
  (910004, 0, 1, 1004, '跨标签话题示例', 'markdown', '同时属于英超和欧冠', NULL, NULL, 0, false, 0, false, 0, 15, 3, 2, 0, 1710001300, 2012, 'seed', '127.0.0.1', 'Local', 1710000400, NULL),
  (910005, 0, 1, 1005, '零评论德甲话题', 'markdown', '用于验证0评论标签', NULL, NULL, 0, false, 0, false, 0, 5, 0, 0, 0, 0, 0, 'seed', '127.0.0.1', 'Local', 1710000500, NULL)
ON CONFLICT (id) DO NOTHING;

-- 3. 话题标签关系
INSERT INTO t_topic_tag (id, topic_id, tag_id, status, last_comment_time, last_comment_user_id, create_time) VALUES
  (920001, 910001, 900001, 0, 1710001000, 2004, 1710000100),
  (920002, 910002, 900002, 0, 1710001100, 2006, 1710000200),
  (920003, 910003, 900003, 0, 1710001200, 2009, 1710000300),
  (920004, 910004, 900001, 0, 1710001300, 2012, 1710000400),
  (920005, 910004, 900003, 0, 1710001300, 2012, 1710000400),
  (920006, 910005, 900004, 0, 0, 0, 1710000500)
ON CONFLICT (id) DO NOTHING;

-- 4. 评论
INSERT INTO t_comment (
  id, user_id, entity_type, entity_id, content, image_list, content_type, quote_id,
  like_count, comment_count, user_agent, ip, ip_location, status, create_time
) VALUES
  (930001, 2001, 'topic', 910001, '英超评论1', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001001),
  (930002, 2002, 'topic', 910001, '英超评论2', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001002),
  (930003, 2003, 'topic', 910001, '英超评论3', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001003),
  (930004, 2004, 'topic', 910001, '英超评论4', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001004),
  (930005, 2005, 'topic', 910002, '西甲评论1', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001101),
  (930006, 2006, 'topic', 910002, '西甲评论2', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001102),
  (930007, 2007, 'topic', 910003, '欧冠评论1', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001201),
  (930008, 2008, 'topic', 910003, '欧冠评论2', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001202),
  (930009, 2009, 'topic', 910003, '欧冠评论3', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001203),
  (930010, 2010, 'topic', 910003, '欧冠评论4', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001204),
  (930011, 2011, 'topic', 910003, '欧冠评论5', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001205),
  (930012, 2012, 'topic', 910004, '跨标签评论1', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001301),
  (930013, 2013, 'topic', 910004, '跨标签评论2', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001302),
  (930014, 2014, 'topic', 910004, '跨标签评论3', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001303),
  -- 非 topic 评论，不应计入统计
  (930015, 2015, 'article', 910001, '这条不应计入', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 0, 1710001401),
  -- 已删除评论，不应计入统计
  (930016, 2016, 'topic', 910002, '已删除评论不计入', NULL, 'markdown', 0, 0, 0, 'seed', '127.0.0.1', 'Local', 1, 1710001402)
ON CONFLICT (id) DO NOTHING;

COMMIT;

-- 预期统计结果：
-- 英超(900001) = 4 + 3 = 7
-- 欧冠(900003) = 5 + 3 = 8
-- 西甲(900002) = 2
-- 德甲(900004) = 0
