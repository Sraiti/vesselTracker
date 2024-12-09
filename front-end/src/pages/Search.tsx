import { useState, useEffect, useRef } from "react";
import Map, { Marker, Source, Layer, MapRef } from "react-map-gl";
import "mapbox-gl/dist/mapbox-gl.css";
import {
  Search as SearchIcon,
  Anchor,
  Ship,
  Compass,
  ShipWheel,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import LocationAutocomplete from "@/components/LocationAutocomplete";
import { Feature, Location, VesselGeoData } from "../types/location";
import { ScheduleGrid } from "@/components/ScheduleGrid";
import { ScheduleResponse } from "@/types/schedule";
import mapboxgl from "mapbox-gl";
import { seaRoute } from "searoute-ts";
import { Schedule } from "@/types/schedule";
import { calculateDistance } from "@/helpers/location";
import { format } from "date-fns";
import { cn } from "@/lib/utils";
import EmptyStateAnimation from "@/components/EmptyStateAnimation";

export default function Search() {
  const [origin, setOrigin] = useState<Location | null>(null);
  const [data, setData] = useState<ScheduleResponse | null>(null);
  const [destination, setDestination] = useState<Location | null>(null);
  //   const [viewState, setViewState] = useState({
  //     longitude: -0.1276,
  //     latitude: 51.5074,
  //     zoom: 2,
  //   });
  const [routeGeometry, setRouteGeometry] = useState<GeoJSON.Feature | null>(
    null
  );
  const [routeDistance, setRouteDistance] = useState<number | null>(null);
  const [selectedSchedule, setSelectedSchedule] = useState<Schedule | null>(
    null
  );
  const [isSearching, setIsSearching] = useState(false);

  const mapRef = useRef<MapRef>(null);
  const mapContainerRef = useRef();

  const handleSearch = async () => {
    setIsSearching(true);
    setData(null);
    if (!origin || !destination) return;
    try {
      const response = await fetch("http://localhost:3058/search", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          OriginPortUnLoCode: origin.unlocode,
          DestinationPortUnLoCode: destination.unlocode,
          Origin: origin.name,
          Destination: destination.name,
          DepartureDate: new Date().toISOString(),
        }),
      });

      const data = await response.json();
      setData(data);

      // Handle the response data
      console.log(data);
    } catch (error) {
      setData(null);
      console.error("Error searching routes:", error);
    } finally {
      setIsSearching(false);
    }
  };

  // Function to fit map view to show both markers
  const fitMapToMarkers = () => {
    if (origin?.location && destination?.location) {
      const bounds = new mapboxgl.LngLatBounds()
        .extend(origin.location)
        .extend(destination.location);
    }
  };

  // Function to create a route line between two points
  const createRouteLine = (start: [number, number], end: [number, number]) => {
    return {
      type: "Feature",
      properties: {},
      geometry: {
        type: "LineString",
        coordinates: [
          [start[1], start[0]], // Convert to [longitude, latitude]
          [end[1], end[0]],
        ],
      },
    } as GeoJSON.Feature;
  };

  // Update viewState when locations are selected
  const setAndFlyToInput = (location: [number, number] | null) => {
    if (location) {
      console.log("location", location);
      console.log({ check: location.length === 2 && location.every((l) => l) });

      if (location.length === 2 && location.every((l) => l)) {
        console.log("valid location");

        if (mapRef.current) {
          console.log("FLY TO :", [location[1], location[0]]);

          mapRef.current.flyTo({
            center: [location[1], location[0]],
            zoom: 4,
            duration: 2000,
            essential: true,
          });
        }

        // setViewState((prev) => ({
        //   ...prev,
        //   longitude: location.location ? location.location[1] : 0,
        //   latitude: location.location ? location.location[0] : 0,
        //   zoom: 1,
        // }));
      }
    }
  };

  // Update route when origin or destination changes
  useEffect(() => {
    if (
      origin?.location?.length === 2 &&
      destination?.location?.length === 2 &&
      origin?.location?.every((l) => l) &&
      destination?.location?.every((l) => l)
    ) {
      try {
        // Create GeoJSON points
        const originPoint = {
          type: "Feature",
          properties: {},
          geometry: {
            type: "Point",
            coordinates: [origin.location[1], origin.location[0]],
          },
        };

        const destinationPoint = {
          type: "Feature",
          properties: {},
          geometry: {
            type: "Point",
            coordinates: [destination.location[1], destination.location[0]],
          },
        };

        // Calculate maritime route
        const route = seaRoute(originPoint, destinationPoint, "kilometers");

        console.log(route);
        setRouteGeometry(route);
        setRouteDistance(route.properties?.length || null);
        fitMapToMarkers();
      } catch (error) {
        console.error("Error calculating maritime route:", error);
        // Fallback to direct line if routing fails
        setRouteGeometry(
          createRouteLine(origin.location, destination.location)
        );
        setRouteDistance(null);
      }
    } else {
      setRouteGeometry(null);
      setRouteDistance(null);
    }
  }, [origin, destination]);

  // Handler for schedule selection
  const handleScheduleSelect = (schedule: Schedule) => {
    setSelectedSchedule(schedule === selectedSchedule ? null : schedule);

    const location = [
      schedule.LastKnownPosition[1],
      schedule.LastKnownPosition[0],
    ] as [number, number];

    console.log({ name: schedule.DepartureVesselName });

    console.log("vessel", { location });

    setAndFlyToInput(location);

    drawVesselRoute(schedule.DepartureVesselMMSI);
  };

  const drawVesselRoute = async (mmsi: string) => {
    try {
      const response = await fetch(
        `http://localhost:3058/vessels/route/geojson?mmsi=${mmsi}`
      );
      const geojson: VesselGeoData = await response.json();

      if (!geojson.features) return;

      const features = geojson.features.filter(
        (feature) => feature.geometry?.coordinates
      );

      // Create points GeoJSON
      const pointsGeoJSON: GeoJSON.FeatureCollection = {
        type: "FeatureCollection",
        features: features.map(
          (feature) =>
            ({
              type: "Feature",
              properties: {},
              geometry: {
                type: "Point",
                coordinates: feature.geometry?.coordinates,
              },
            } as any)
        ),
      };

      // Create route as before
      const allRouteSegments: [number, number][] = [];
      for (let i = 0; i < features.length - 1; i++) {
        const coordinates = features[i]?.geometry?.coordinates;
        const nextCoordinates = features[i + 1]?.geometry?.coordinates;

        if (!coordinates || !nextCoordinates) continue;

        // Calculate distance between points
        const distance = calculateDistance(
          coordinates[1],
          coordinates[0],
          nextCoordinates[1],
          nextCoordinates[0]
        );

        // If points are too close (less than 5km), just use direct line
        if (distance < 5) {
          // For very close points, use linear interpolation
          const midPoint = [
            (coordinates[0] + nextCoordinates[0]) / 2,
            (coordinates[1] + nextCoordinates[1]) / 2,
          ];
          allRouteSegments.push(
            coordinates as [number, number],
            midPoint as [number, number]
          );
          continue;
        }

        // For longer distances, use sea route calculation
        const start = {
          type: "Feature",
          properties: {},
          geometry: {
            type: "Point",
            coordinates: coordinates,
          },
        };

        const end = {
          type: "Feature",
          properties: {},
          geometry: {
            type: "Point",
            coordinates: nextCoordinates,
          },
        };

        try {
          const route = seaRoute(start, end, "miles");

          if (route?.geometry?.coordinates) {
            // Add some interpolation points between segments
            const routeCoords = route.geometry.coordinates;
            for (let j = 0; j < routeCoords.length - 1; j++) {
              allRouteSegments.push(routeCoords[j] as [number, number]);
              // Add interpolated point between segments
              if (j < routeCoords.length - 1) {
                const interpolated = [
                  (routeCoords[j][0] + routeCoords[j + 1][0]) / 2,
                  (routeCoords[j][1] + routeCoords[j + 1][1]) / 2,
                ];
                allRouteSegments.push(interpolated as [number, number]);
              }
            }
            // Add final point
            allRouteSegments.push(
              routeCoords[routeCoords.length - 1] as [number, number]
            );
          }
        } catch (error) {
          console.error("Error calculating route:", error);
          allRouteSegments.push(coordinates as [number, number]);
        }
      }

      const routeLine: GeoJSON.Feature = {
        type: "Feature",
        properties: {},
        geometry: {
          type: "LineString",
          coordinates: allRouteSegments || [],
        },
      };

      const map = mapRef.current?.getMap();
      if (map) {
        // Remove existing layers and sources
        if (map.getSource("vessel-route")) {
          map.removeLayer("vessel-route-layer");
          map.removeSource("vessel-route");
        }
        if (map.getSource("vessel-points")) {
          map.removeLayer("vessel-points-layer");
          map.removeSource("vessel-points");
        }

        // Add route source and layer
        map.addSource("vessel-route", {
          type: "geojson",
          data: routeLine,
        });

        map.addLayer({
          id: "vessel-route-layer",
          type: "line",
          source: "vessel-route",
          paint: {
            "line-color": "#fef9c3",
            "line-width": 2,
            "line-dasharray": [2, 1],
          },
        });

        // Add points source and layer
        map.addSource("vessel-points", {
          type: "geojson",
          data: pointsGeoJSON,
        });

        map.addLayer({
          id: "vessel-points-layer",
          type: "circle",
          source: "vessel-points",
          paint: {
            "circle-radius": 3,
            "circle-color": "#fef9c3",
          },
        });

        // Fit map to bounds
        const bounds = allRouteSegments.reduce(
          (bounds: any, coord: number[]) => {
            return bounds.extend(coord);
          },
          new mapboxgl.LngLatBounds(allRouteSegments[0], allRouteSegments[0])
        );

        map.fitBounds(bounds, {
          padding: 50,
        });

        // After adding vessel route layers, update the opacity of the original route
        if (map.getLayer("route")) {
          map.setPaintProperty("route", "line-color", "#94a3b8");
          map.setPaintProperty("route", "line-opacity", 0.3); // Make original route more transparent
        }
      }
    } catch (error) {
      console.error("Error fetching vessel route:", error);
    }
  };

  return (
    <div
      className="h-screen w-screen relative overflow-hidden"
      id="map-container"
      ref={mapContainerRef.current}
    >
      <Map
        ref={mapRef}
        mapboxAccessToken={import.meta.env.VITE_MAPBOX_TOKEN}
        initialViewState={{
          longitude: -6,
          latitude: 32,
          zoom: 2.7,
        }}
        style={{ width: "100%", height: "100%" }}
        mapStyle="mapbox://styles/mapbox/dark-v11"
        onLoad={(event) => {
          //   const map = event.target;
          //   // Add custom layer to merge Western Sahara with Morocco
          //   const WORLD_VIEW = "MA";
          //   const adminLayers = [
          //     "admin-0-boundary",
          //     //   "admin-1-boundary",
          //     "admin-0-boundary-disputed",
          //     //   "admin-1-boundary-bg",
          //     //   "admin-0-boundary-bg",
          //     "country-label",
          //   ];
          //   adminLayers.forEach((adminLayer) => {
          //     map.setFilter(adminLayer, [
          //       "match",
          //       ["get", "worldview"],
          //       ["all", WORLD_VIEW],
          //       true,
          //       false,
          //     ]);
          //   });
        }}
      >
        {routeGeometry && (
          <Source type="geojson" data={routeGeometry}>
            <Layer
              id="route"
              type="line"
              paint={{
                "line-color": "#94a3b8",
                "line-width": 2,
                "line-dasharray": [2, 1],
                "line-opacity": 1, // Add default opacity
              }}
            />
          </Source>
        )}

        {origin?.location &&
          origin.location.length === 2 &&
          origin.location.every((l) => l) && (
            <Marker
              longitude={origin.location[1]}
              latitude={origin.location[0]}
              anchor="center"
            >
              <div className="relative group">
                {selectedSchedule?.DepartureDateTime && (
                  <div className="absolute left-1/2 -translate-x-1/2 -top-7 bg-background/95 p-1 rounded shadow-lg whitespace-nowrap text-xs">
                    {format(
                      new Date(selectedSchedule?.DepartureDateTime),
                      "PPP"
                    )}
                  </div>
                )}
                <div className="text-primary bg-background/95 p-2 rounded-full shadow-lg">
                  <Anchor className="h-6 w-6" />
                </div>
              </div>
            </Marker>
          )}

        {destination?.location &&
          destination.location.length === 2 &&
          destination.location.every((l) => l) && (
            <Marker
              longitude={destination.location[1]}
              latitude={destination.location[0]}
              anchor="center"
            >
              <div className="relative group">
                {selectedSchedule?.ArrivalDateTime && (
                  <div className="absolute left-1/2 -translate-x-1/2 -top-7 bg-background/95 p-1 rounded shadow-lg whitespace-nowrap text-xs">
                    {format(new Date(selectedSchedule?.ArrivalDateTime), "PPP")}
                  </div>
                )}
                <div className="text-destructive bg-background/95 p-2 rounded-full shadow-lg m-1">
                  <Anchor className="h-6 w-6" />
                </div>
              </div>
            </Marker>
          )}

        {routeDistance && (
          <div className="absolute bottom-4 right-4 bg-background/95 p-4 rounded-lg shadow-lg">
            <h3 className="font-semibold">Maritime Route</h3>
            <p>Distance: {Math.round(routeDistance)} nautical miles</p>
            <p>
              Est. Duration: {Math.round((routeDistance / 20) * 24)} hours
              {/* Assuming average vessel speed of 20 knots */}
            </p>
          </div>
        )}

        {/* Add vessel position marker */}
        {selectedSchedule?.LastKnownPosition &&
          selectedSchedule.LastKnownPosition.length === 2 &&
          selectedSchedule.LastKnownPosition.every((l) => l) && (
            <Marker
              longitude={selectedSchedule.LastKnownPosition[0]}
              latitude={selectedSchedule.LastKnownPosition[1]}
              anchor="center"
            >
              <div className="relative group">
                {/* Ripple effect */}
                <div className="absolute -inset-2 bg-yellow-500/20 rounded-full animate-ping" />
                <div className="absolute -inset-2 bg-yellow-500/40 rounded-full" />
                {/* Vessel icon */}
                <div className="relative bg-background text-yellow-500 p-2 rounded-full shadow-lg transform transition-transform group-hover:scale-110 cursor-pointer">
                  <Ship className="h-6 w-6 animate-[wiggle_4s_ease-in-out_infinite]" />
                </div>
                {/* Hover tooltip */}
                <div className="absolute left-1/2 -translate-x-1/2 -top-12 bg-background/95 p-2 rounded shadow-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap">
                  {selectedSchedule.DepartureVesselName}
                </div>
              </div>
            </Marker>
          )}
      </Map>

      <div className="absolute max-h-screen w-1/3 h-full left-0 top-0 mt-1">
        <div className="backdrop-blur rounded-lg shadow-lg flex flex-col items-center">
          <div className="flex flex-col gap-3 p-8 w-full">
            <LocationAutocomplete
              value={origin}
              onChange={(value) => {
                setOrigin(value);
                setAndFlyToInput(value?.location || null);
              }}
              placeholder="Origin port"
            />

            <LocationAutocomplete
              value={destination}
              onChange={(value) => {
                setDestination(value);
                setAndFlyToInput(value?.location || null);
              }}
              placeholder="Destination port"
            />
          </div>
          <Button
            className={cn(
              "m-2 p-2",
              "bg-background/95 hover:bg-background text-primary",
              "transition-all duration-300 ease-in-out",
              "group relative overflow-hidden shadow-lg flex items-center gap-2",
              isSearching && "cursor-wait"
            )}
            onClick={() => {
              handleSearch();
              if (origin?.location && destination?.location) {
                const bounds = new mapboxgl.LngLatBounds(
                  [origin.location[1], origin.location[0]],
                  [destination.location[1], destination.location[0]]
                );
                mapRef.current?.fitBounds(bounds, {
                  padding: 100,
                  duration: 1000,
                });
              }
            }}
            title="Find Routes"
            aria-label="Find Routes"
            disabled={!origin || !destination || isSearching}
          >
            <ShipWheel
              className={cn(
                "h-5 w-5 transition-transform duration-700 ease-in-out text-gray-500",
                "group-hover:rotate-180",
                isSearching && "animate-spin"
              )}
            />
            <span>Find Routes</span>
            <span className="sr-only">
              {isSearching ? "Searching..." : "Find Routes"}
            </span>
          </Button>
        </div>
        {isSearching && (
          <div className="backdrop-blur rounded-lg shadow-lg mt-4 overflow-hidden flex justify-center items-center w-full">
            <EmptyStateAnimation isLoading={isSearching} />
          </div>
        )}
        {data && (
          <div className="backdrop-blur rounded-lg shadow-lg overflow-y-auto max-h-[calc(100vh-200px)] scrollbar-hidden">
            <ScheduleGrid data={data} onSelect={handleScheduleSelect} />
          </div>
        )}
      </div>

      {selectedSchedule?.LastKnownPosition &&
        selectedSchedule.LastKnownPosition.length === 2 &&
        selectedSchedule.LastKnownPosition.every((l) => l) && (
          <div className="absolute top-4 right-4 bg-background/95 p-4 rounded-lg shadow-lg">
            <h3 className="font-semibold">Vessel Position</h3>
            <p>Vessel: {selectedSchedule.DepartureVesselName}</p>
            <p>IMO: {selectedSchedule.DepartureVesselIMONumber}</p>
            <p>MMSI: {selectedSchedule.DepartureVesselMMSI}</p>
            <p>
              Position: {selectedSchedule.LastKnownPosition[1].toFixed(4)}°N,{" "}
              {selectedSchedule.LastKnownPosition[0].toFixed(4)}°E
            </p>
          </div>
        )}
    </div>
  );
}
