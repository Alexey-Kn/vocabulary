package ui

import "time"

const (
	//Protection of "blinking" elements while they are disabled for too short period.
	MIN_TIME_OF_WAITING_SCREEN_DISPLAYING = time.Millisecond * 2000

	//The time period to wait before disabling all the widgets excluding "Main menu" button
	//(indication that the program got stuck and waiting for some remote resource answer).
	TIME_BEFORE_SHOWING_WAITING_SCREEN = time.Millisecond * 1000

	//The delay before switching to the next task after successfull solution of current one.
	//Souldn't be bigger than TIME_BEFORE_SHOWING_WAITING_SCREEN.
	TIME_TO_DEMONSTRATE_RIGHT_ANSWER = time.Millisecond * 750
)
