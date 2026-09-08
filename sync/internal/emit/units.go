// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// units.go is the symbol table: how VSS writes each unit.
//
// Kept beside the vocabulary emitter because both readers of it are here --
// the Unit enum's comments and the "Unit: km/h." line on every signal field.
// A second copy of this mapping is how a field comment and the enum it refers
// to end up disagreeing.

var unitSymbolTable = map[string]string{
	"MILLIMETER": "mm", "CENTIMETER": "cm", "METER": "m", "KILOMETER": "km",
	"INCH": "in", "KILOMETER_PER_HOUR": "km/h", "METERS_PER_SECOND": "m/s",
	"METERS_PER_SECOND_SQUARED": "m/s^2", "CENTIMETERS_PER_SECOND_SQUARED": "cm/s^2",
	"MILLILITER": "ml", "LITER": "l", "CUBIC_CENTIMETERS": "cm^3",
	"DEGREE_CELSIUS": "degC", "DEGREE_FAHRENHEIT": "degF", "DEGREE": "deg",
	"DEGREE_PER_SECOND": "deg/s", "RADIANS_PER_SECOND": "rad/s",
	"WATT": "W", "KILOWATT": "kW", "HORSEPOWER": "PS", "KILOWATT_HOURS": "kWh",
	"GRAM": "g", "KILOGRAM": "kg", "POUND": "lbs", "VOLT": "V",
	"AMPERE": "A", "AMPERE_HOURS": "Ah", "NANOSECOND": "ns", "MILLISECOND": "ms",
	"SECOND": "s", "MINUTE": "min", "HOUR": "h", "DAYS": "day",
	"WEEKS": "weeks", "MONTHS": "months", "YEARS": "years",
	"U_N_I_X_TIMESTAMP": "UNIX timestamp", "ISO_8601": "ISO 8601",
	"MILLIBAR": "mbar", "PASCAL": "Pa", "KILOPASCAL": "kPa",
	"POUNDS_PER_SQUARE_INCH": "psi", "STARS": "stars",
	"GRAMS_PER_SECOND": "g/s", "GRAMS_PER_KILOMETER": "g/km",
	"KILOWATT_HOURS_PER_100_KILOMETERS": "kWh/100km", "WATT_HOUR_PER_KM": "Wh/km",
	"MILLILITER_PER_100_KILOMETERS": "ml/100km", "LITER_PER_100_KILOMETERS": "l/100km",
	"LITER_PER_HOUR": "l/h", "MILES_PER_US_GALLON": "mpg (US)",
	"MILES_PER_IMPERIAL_GALLON":      "mpg (imperial)",
	"MILES_PER_US_GALLON_EQUIVALENT": "MPGe",
	"MILES_PER_US_GALLON_DEPRECATED": "mpg", "KILOMETERS_PER_LITER": "km/l",
	"NEWTON": "N", "KILO_NEWTON": "kN", "NEWTON_METER": "Nm",
	"REVOLUTIONS_PER_MINUTE": "rpm", "HERTZ": "Hz", "CYCLES_PER_MINUTE": "cpm",
	"BEATS_PER_MINUTE": "bpm", "RATIO": "ratio", "PERCENT": "percent",
	"NANO_METER_PER_KILOMETER": "nm/km", "DECIBEL_MILLIWATT": "dBm",
	"DECIBEL": "dB", "OHM": "Ohm", "LUX": "lx",
}

var unitNotes = map[string]string{
	"U_N_I_X_TIMESTAMP": "The source model spells this `U_N_I_X_TIMESTAMP`, an " +
		"artefact of splitting the VSS name \"UNIX Timestamp\" on case.",
	"MILES_PER_US_GALLON_DEPRECATED": "VSS retains this alongside " +
		"UNIT_MILES_PER_US_GALLON for compatibility; prefer that one.",
}
