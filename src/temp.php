<?php
$db = new mysqli('localhost', 'id1678131_vvestin', 'W428056w', 'id1678131_pinturella');
$words = "peak to peak,geek to freak,hankla,lehr,flanhofer,physical education,rubiks cube,beep sheep,scantily clad calculator,sixty two,ash ketchum,pikachu,jigglypuff,lucario,stunfisk,snorlax,charizard,blastoise,venasaur,greninja,ti-84,waystoner,animorphs,vegimorphs,brawlhalla,goats,corsair rgb";
$words = explode(',', $words);

foreach ($words as $word) {
  $db->query("INSERT INTO `words` VALUES ('$word')");
  echo($word.'<br>');
}


?>
