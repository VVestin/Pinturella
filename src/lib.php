<?php
define('DB_HOST', 'localhost');
define('DB_USER', 'id1678131_vvestin');
define('DB_PASSWORD', 'W428056w');
define('DB_DATABASE', 'id1678131_pinturella');

set_error_handler('pinturellaErrorHandler', E_ALL);

function pinturellaErrorHandler($code, $message, $file, $lineNumber) {
	if (ob_get_length()) ob_clean();
	$errorMessage = 'Error: ' . $code . chr(10) .
						 'Message: ' . $message . chr(10) .
					 	 'File: ' . $file . chr(10) .
						 'Line: ' . $lineNumber;
	echo $errorMessage;
	exit;
}

function connectToDB() {
   return new mysqli(DB_HOST, DB_USER, DB_PASSWORD, DB_DATABASE);
}

?>
