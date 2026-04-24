package essence

/*
#cgo LDFLAGS: -L../../../../heartbeat-project/target/release -lessence_core
#include <stdlib.h>
#include <stdint.h>

// Rust FFI definitions
typedef struct {
    char* user_id;
    double distance_km;
} FFIUserMatch;

extern void* essence_new();
extern void essence_free(void* db_ptr);
extern void essence_update_location(void* db_ptr, const char* user_id, double lat, double lon, double alt, uint64_t timestamp);
extern int essence_get_nearby_users(void* db_ptr, const char* user_id, double radius_km, FFIUserMatch* out_matches, int max_matches);
extern void essence_free_matches(FFIUserMatch* matches, int count);
*/
import "C"
import (
	"time"
	"unsafe"
)

// Match represents a nearby user match
type Match struct {
	UserID     string
	DistanceKM float64
}

// EssenceDB is a Go wrapper for the Rust Essence database
type EssenceDB struct {
	ptr unsafe.Pointer
}

// New creates a new EssenceDB instance
func New() *EssenceDB {
	return &EssenceDB{
		ptr: C.essence_new(),
	}
}

// Close frees the Rust memory
func (db *EssenceDB) Close() {
	if db.ptr != nil {
		C.essence_free(db.ptr)
		db.ptr = nil
	}
}

// UpdateLocation updates a user's location
func (db *EssenceDB) UpdateLocation(userID string, lat, lon, alt float64) {
	cUser := C.CString(userID)
	defer C.free(unsafe.Pointer(cUser))
	
	timestamp := C.uint64_t(time.Now().Unix())
	C.essence_update_location(db.ptr, cUser, C.double(lat), C.double(lon), C.double(alt), timestamp)
}

// GetNearbyUsers returns users within a radius
func (db *EssenceDB) GetNearbyUsers(userID string, radiusKm float64, max int) []Match {
	cUser := C.CString(userID)
	defer C.free(unsafe.Pointer(cUser))
	
	outMatches := make([]C.FFIUserMatch, max)
	
	found := C.essence_get_nearby_users(
		db.ptr,
		cUser,
		C.double(radiusKm),
		(*C.FFIUserMatch)(unsafe.Pointer(&outMatches[0])),
		C.int(max),
	)
	
	defer C.essence_free_matches((*C.FFIUserMatch)(unsafe.Pointer(&outMatches[0])), found)
	
	results := make([]Match, 0, int(found))
	for i := 0; i < int(found); i++ {
		results = append(results, Match{
			UserID:     C.GoString(outMatches[i].user_id),
			DistanceKM: float64(outMatches[i].distance_km),
		})
	}
	
	return results
}
