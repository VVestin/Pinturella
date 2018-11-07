<?php
$db = new mysqli('localhost', 'id1678131_vvestin', 'W428056w', 'id1678131_pinturella');
$word = $db->query("SELECT * FROM `words` ORDER BY RAND() LIMIT 1");
echo $word->fetch_object()->word;

?>
