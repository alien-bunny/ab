// Copyright 2018 Tamás Demeter-Haludka
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package ab is the main package of the Alien Bunny web development kit.

This package contains the server and the middlewares of the framework. If you want to get started, you probably want to take a look at Hop and PetBunny.

The lowest level component is the Server component. It is a wrapper on the top of httprouter that adds middlewares along with a few useful features. On the server you can configure Services. Services are logical units of endpoints that share a piece of schema. On the top of the services, there are resources. Resources are CRUD endpoints. There are delegates and event handlers that help augmenting the functionality of the ResourceController.

Entities are a pointer to a struct that can be stored in a database. EntityController automatically does CRUD on entities, and the operations can be customized with delegates and event handlers.

EntityResource combines the EntityController and the ResourceController to easily expose an entity through API endpoints.

Quick and dirty usage:

	func main() {
		ab.Hop(func(cfg *viper.Viper, s *ab.Server) error {
			ec := ab.NewEntityController(s.GetDBConnection())
			ec.Add(&Content{}, contentEntityDelegate{})

			res := ab.EntityResource(ec, &Content{}, ab.EntityResourceConfig{
				DisableList: true,
				DisablePost: true,
				DisablePut: true,
				DisableDelete: true,
			})

			s.RegisterService(res)

			return nil
		}, nil)
	}
*/
package ab
